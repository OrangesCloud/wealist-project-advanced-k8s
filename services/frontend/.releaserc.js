/**
 * Semantic Release Configuration for Frontend
 *
 * 커밋 메시지 형식:
 *   - front-fix: 버그 수정 → patch (1.0.0 → 1.0.1)
 *   - front-feat: 새 기능 → minor (1.0.0 → 1.1.0)
 *   - front-feat!: Breaking Change → major (1.0.0 → 2.0.0)
 *   - front-fix!: Breaking Change → major
 *   - front-docs: 문서 수정 → 버전업 없음
 *   - front-style: 스타일 수정 → 버전업 없음
 *   - front-refactor: 리팩토링 → 버전업 없음
 *   - front-perf: 성능 개선 → patch
 *   - front-test: 테스트 추가 → 버전업 없음
 *
 * 예시:
 *   front-fix: 로그인 버튼 클릭 안되는 문제 수정
 *   front-feat: 다크모드 기능 추가
 *   front-feat!: API 응답 구조 변경 (Breaking Change)
 */

module.exports = {
  branches: ['main', 'dev', { name: 'dev-frontend', prerelease: 'dev' }],
  tagFormat: 'frontend-v${version}',

  plugins: [
    // 1. 커밋 분석 (front- prefix만 인식)
    [
      '@semantic-release/commit-analyzer',
      {
        preset: 'conventionalcommits',
        parserOpts: {
          // front-feat:, front-fix: 형식 파싱
          headerPattern: /^front-(\w+)(?:\((.+)\))?!?:\s(.+)$/,
          headerCorrespondence: ['type', 'scope', 'subject'],
          noteKeywords: ['BREAKING CHANGE', 'BREAKING-CHANGE'],
        },
        releaseRules: [
          // Breaking changes
          { type: 'feat', breaking: true, release: 'major' },
          { type: 'fix', breaking: true, release: 'major' },
          // Features
          { type: 'feat', release: 'minor' },
          // Bug fixes & patches
          { type: 'fix', release: 'patch' },
          { type: 'perf', release: 'patch' },
          // No release
          { type: 'docs', release: false },
          { type: 'style', release: false },
          { type: 'refactor', release: false },
          { type: 'test', release: false },
          { type: 'chore', release: false },
          { type: 'ci', release: false },
        ],
      },
    ],

    // 2. 릴리스 노트 생성
    [
      '@semantic-release/release-notes-generator',
      {
        preset: 'conventionalcommits',
        parserOpts: {
          headerPattern: /^front-(\w+)(?:\((.+)\))?!?:\s(.+)$/,
          headerCorrespondence: ['type', 'scope', 'subject'],
        },
        presetConfig: {
          types: [
            { type: 'feat', section: '✨ Features', hidden: false },
            { type: 'fix', section: '🐛 Bug Fixes', hidden: false },
            { type: 'perf', section: '⚡ Performance', hidden: false },
            { type: 'docs', section: '📚 Documentation', hidden: true },
            { type: 'style', section: '💄 Styles', hidden: true },
            { type: 'refactor', section: '♻️ Refactoring', hidden: true },
            { type: 'test', section: '✅ Tests', hidden: true },
            { type: 'chore', section: '🔧 Chores', hidden: true },
            { type: 'ci', section: '👷 CI', hidden: true },
          ],
        },
      },
    ],

    // 3. package.json 버전 업데이트
    '@semantic-release/npm',

    // 4. CHANGELOG.md 생성
    [
      '@semantic-release/changelog',
      {
        changelogFile: 'CHANGELOG.md',
      },
    ],

    // 5. Git 커밋 & 태그
    [
      '@semantic-release/git',
      {
        assets: ['package.json', 'CHANGELOG.md'],
        message: 'chore(frontend): release ${nextRelease.version} [skip ci]\n\n${nextRelease.notes}',
      },
    ],

    // 6. GitHub Release
    '@semantic-release/github',
  ],
};
