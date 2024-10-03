#!/usr/bin/env node

import { readFileSync, writeFileSync } from 'fs'
import { homedir } from 'os'
import { join } from 'path'
import { select, confirm } from '@inquirer/prompts'
import { execSync, spawnSync } from 'child_process'
import { Command } from 'commander'
import Fuse from 'fuse.js'

interface AWSProfile {
  name: string
  aws_access_key_id?: string
  aws_secret_access_key?: string
  region?: string
  role_arn?: string
  source_profile?: string
}

const getProfileEmoji = (profileName: string): string => {
  if (profileName.includes('prod')) return '🔴'
  if (profileName.includes('test')) return '🟡'
  return '🟢'
}
const LAST_USED_FILE = join(homedir(), '.aws-profile-selector-last')

const isValidProfileName = (name: string): boolean => {
  const profileNameRegex = /^[a-zA-Z0-9][a-zA-Z0-9_-]*$/
  return profileNameRegex.test(name)
}

const parseAWSCredentials = (content: string): Record<string, AWSProfile> => {
  const profiles: Record<string, AWSProfile> = {}
  let currentProfile: string | null = null

  content.split('\n').forEach((line) => {
    const trimmedLine = line.trim()
    if (trimmedLine.startsWith('[') && trimmedLine.endsWith(']')) {
      const profileName = trimmedLine.slice(1, -1)
      if (isValidProfileName(profileName) && profileName !== 'default') {
        currentProfile = profileName
        profiles[currentProfile] = { name: currentProfile }
      } else {
        currentProfile = null
      }
    } else if (currentProfile && trimmedLine.includes('=')) {
      const [key, value] = trimmedLine.split('=').map((s) => s.trim())
      profiles[currentProfile][key as keyof AWSProfile] = value
    }
  })

  return profiles
}

const getLastUsedProfile = (): string | null => {
  try {
    return readFileSync(LAST_USED_FILE, 'utf-8').trim()
  } catch {
    return null
  }
}

const saveLastUsedProfile = (profileName: string): void => {
  writeFileSync(LAST_USED_FILE, profileName)
}

const getCurrentRegion = (): string => {
  const result = spawnSync('aws', ['configure', 'get', 'region'], {
    encoding: 'utf-8',
  })
  return result.stdout.trim() || 'Not set'
}

const fuzzySearchProfiles = (
  query: string,
  profiles: Record<string, AWSProfile>
): AWSProfile[] => {
  const fuse = new Fuse(Object.values(profiles), {
    keys: ['name'],
    threshold: 0.4,
  })
  return fuse.search(query).map((result) => result.item)
}

const main = async () => {
  const program = new Command()

  program
    .name('aws-profile-selector')
    .description('CLI to select AWS profiles')
    .argument('[search]', 'Fuzzy search for a profile')
    .action(async (search?: string) => {
      try {
        const credentialsPath = join(homedir(), '.aws', 'credentials')
        const credentialsContent = readFileSync(credentialsPath, 'utf-8')
        const profiles = parseAWSCredentials(credentialsContent)

        if (search) {
          const searchResults = fuzzySearchProfiles(search, profiles)
          if (searchResults.length > 0) {
            const suggestedProfile = searchResults[0]
            const useProfile = await confirm({
              message: `Use suggested profile "${suggestedProfile.name}"?`,
            })
            if (useProfile) {
              selectAndUseProfile(suggestedProfile.name)
              return
            }
          } else {
            console.log('No matching profiles found.')
          }
        }

        await showProfileSelectionPrompt(profiles)
      } catch (error: any) {
        console.error('An error occurred:', error.message)
        process.exit(1)
      }
    })

  program.parse()
}

const showProfileSelectionPrompt = async (
  profiles: Record<string, AWSProfile>
) => {
  const profileEntries = Object.entries(profiles)

  if (profileEntries.length === 0) {
    console.log('No non-default profiles found in AWS credentials.')
    process.exit(0)
  }

  const lastUsedProfile = getLastUsedProfile()
  const defaultIndex = lastUsedProfile
    ? profileEntries.findIndex(([name]) => name === lastUsedProfile)
    : 0

  const currentRegion = getCurrentRegion()
  console.log(`Current default region: ${currentRegion}`)

  const selectedProfile = await select({
    message: 'Select an AWS profile:',
    choices: profileEntries.map(([name, profile], index) => {
      const emoji = getProfileEmoji(name)

      return {
        name: `${index + 1}. ${name?.split('-').join(' ')}${
          profile.region ? ` (${profile.region})` : ''
        } ${emoji}`,
        value: name,
        short: `${index + 1}`,
      }
    }),
    default: profileEntries[defaultIndex][0],
    pageSize: 20,
  })

  if (!selectedProfile) {
    console.log('Selection cancelled')
    process.exit(0)
  }

  selectAndUseProfile(selectedProfile)
}

const selectAndUseProfile = (profileName: string) => {
  // Save the selected profile
  saveLastUsedProfile(profileName)

  // Set the AWS_PROFILE environment variable
  // process.env.AWS_PROFILE = profileName

  console.log(`Selected profile: ${profileName}`)

  // Check the new region
  const newRegion = getCurrentRegion()
  console.log(`New default region: ${newRegion}`)

  // Execute AWS CLI command
  try {
    const result = execSync(
      `AWS_PROFILE=${profileName} aws sts get-caller-identity`,
      {
        encoding: 'utf-8',
      }
    )
    console.log('AWS CLI command result:', result)
  } catch (error: any) {
    console.error('Error executing AWS CLI command:', error.message)
  }
}

main()
