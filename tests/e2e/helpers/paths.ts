import { join, resolve } from 'node:path'

// Repo-relative paths, so the tests run from any checkout location (CI uses /__w/...)
export const REPO_ROOT = resolve(__dirname, '../../..')
export const WEB_DIR = join(REPO_ROOT, 'web')
export const BINARY_PATH = join(REPO_ROOT, 'bin', 'jokateko')
