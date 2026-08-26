import { useLang, useT } from '../i18n'

const ISSUE_URL = 'https://github.com/Knights-of-the-Found-Table/MarvelChampionsNext/issues/new'

export default function ReportBugButton({ className = '' }: { className?: string }) {
  const lang = useLang()
  const t = useT()

  function reportBug() {
    const gameId = window.location.pathname.match(/^\/games\/(\d+)/)?.[1] ?? ''
    const context = [
      `- URL: ${window.location.href}`,
      `- Game ID: ${gameId || 'N/A'}`,
      `- Time: ${new Date().toISOString()}`,
      `- Language: ${lang}`,
      `- Browser: ${navigator.userAgent}`,
      `- Viewport: ${window.innerWidth} × ${window.innerHeight}`,
      `- Screen: ${window.screen.width} × ${window.screen.height}`,
    ].join('\n')

    // Issue 模板取自共享目录（reportbug.body，%s 处填自动收集的环境信息）。
    const body = t('reportbug.body', context)

    const params = new URLSearchParams({
      title: '[Bug] ',
      body,
    })
    window.open(`${ISSUE_URL}?${params.toString()}`, '_blank', 'noopener,noreferrer')
  }

  return (
    <button
      type="button"
      className={`bug-report-btn ${className}`.trim()}
      onClick={reportBug}
      title={t('nav.reportBugHint')}
    >
      <span aria-hidden="true">🐞</span>
      <span className="bug-report-label">{t('nav.reportBug')}</span>
    </button>
  )
}
