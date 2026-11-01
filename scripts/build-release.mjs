import fs from 'node:fs'
import path from 'node:path'
import { spawnSync } from 'node:child_process'

const root = process.cwd()
const stripV = (value) => String(value || '').trim().replace(/^v/, '')
const version = stripV(process.env.VERSION || process.env.GITHUB_REF_NAME || '0.0.0')
const repo = String(process.env.REPO || '').trim()
const commit = String(process.env.COMMIT || 'local').trim()
const buildDate = String(process.env.BUILD_DATE || new Date().toISOString()).trim()
const platform = String(process.env.PLATFORM || 'darwin/arm64').trim()
const [goos, goarch] = platform.split('/')
if (!version || !goos || !goarch) {
  console.error('VERSION and PLATFORM (os/arch) are required')
  process.exit(2)
}

const wailsPath = path.join(root, 'wails.json')
const wails = JSON.parse(fs.readFileSync(wailsPath, 'utf8'))
wails.info = wails.info || {}
wails.info.productVersion = version
wails.info.productName = 'NPC 客户端'
fs.writeFileSync(wailsPath, JSON.stringify(wails, null, 2) + '\n')

const ldflags = [
  '-s',
  '-w',
  '-X npc-gui/internal/buildinfo.Version=' + version,
  '-X npc-gui/internal/buildinfo.Commit=' + commit,
  '-X npc-gui/internal/buildinfo.BuildDate=' + buildDate,
]
if (repo) {
  ldflags.push('-X npc-gui/internal/buildinfo.Repo=' + repo)
}

const wailsBin = process.platform === 'win32' ? 'wails.exe' : 'wails'
console.log('Building NPC GUI', version, platform, 'repo=' + (repo || '<dev>'))
const wailsArgs = ['build', '-clean', '-platform', platform, '-ldflags', ldflags.join(' ')]
if (goos === 'linux') {
  wailsArgs.push('-tags', 'webkit2_41')
}
if (goos === 'windows') {
  wailsArgs.push('-nsis')
}
const build = spawnSync(
  wailsBin,
  wailsArgs,
  { cwd: root, stdio: 'inherit', env: process.env },
)
if (build.error) {
  console.error('Unable to run wails:', build.error.message)
  process.exit(1)
}
if (build.status !== 0) {
  process.exit(build.status || 1)
}

const dist = path.join(root, 'dist')
fs.rmSync(dist, { recursive: true, force: true })
fs.mkdirSync(dist, { recursive: true })
const baseName = 'npc-gui_' + version + '_' + goos + '_' + goarch

function run(command, args) {
  const result = spawnSync(command, args, { cwd: root, stdio: 'inherit' })
  if (result.error || result.status !== 0) {
    console.error('Command failed:', command, args.join(' '), result.error?.message || '')
    process.exit(result.status || 1)
  }
}

if (goos === 'darwin') {
  const app = path.join(root, 'build', 'bin', 'npc-gui.app')
  if (!fs.existsSync(app)) {
    console.error('Missing app bundle:', app)
    process.exit(1)
  }
  run('ditto', ['-c', '-k', '--sequesterRsrc', '--keepParent', app, path.join(dist, baseName + '.app.zip')])
} else if (goos === 'windows') {
  const exe = path.join(root, 'build', 'bin', 'npc-gui.exe')
  if (!fs.existsSync(exe)) {
    console.error('Missing executable:', exe)
    process.exit(1)
  }
  fs.copyFileSync(exe, path.join(dist, baseName + '.exe'))
  const binDir = path.join(root, 'build', 'bin')
  const installer = fs.readdirSync(binDir).find((name) => name.endsWith('-installer.exe'))
  if (!installer) {
    console.error('Missing NSIS installer under:', binDir)
    process.exit(1)
  }
  fs.copyFileSync(path.join(binDir, installer), path.join(dist, baseName + '_installer.exe'))
} else {
  const binary = path.join(root, 'build', 'bin', 'npc-gui')
  if (!fs.existsSync(binary)) {
    console.error('Missing executable:', binary)
    process.exit(1)
  }
  run('tar', ['-czf', path.join(dist, baseName + '.tar.gz'), '-C', path.join(root, 'build', 'bin'), 'npc-gui'])
}

for (const name of fs.readdirSync(dist)) {
  const stat = fs.statSync(path.join(dist, name))
  console.log('artifact', name, stat.size + ' bytes')
}
console.log('release assets written to', dist)
