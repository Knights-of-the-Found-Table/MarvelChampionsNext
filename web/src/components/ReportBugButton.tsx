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

    const body = lang === 'zh'
      ? `## 问题描述\n\n请简要说明发生了什么。\n\n## 复现步骤\n\n1. \n2. \n3. \n\n## 预期结果\n\n\n## 实际结果\n\n\n## 截图或录像\n\n可以直接把图片拖进这里。\n\n## 自动收集的环境信息\n\n${context}\n`
      : `## What happened?\n\nPlease describe the problem.\n\n## Steps to reproduce\n\n1. \n2. \n3. \n\n## Expected result\n\n\n## Actual result\n\n\n## Screenshot or recording\n\nDrag images directly into this section.\n\n## Automatically collected context\n\n${context}\n`

    const params = new URLSearchParams({
      title: lang === 'zh' ? '[Bug] ' : '[Bug] ',
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
