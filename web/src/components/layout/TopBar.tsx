import { addMonths, addWeeks, subMonths, subWeeks, startOfToday } from 'date-fns'
import { useCalendarStore } from '../../store/calendarStore'
import { formatMonthYear, formatWeekTitle } from '../../lib/dateUtils'

interface TopBarProps {
  onNewEvent: () => void
}

export default function TopBar({ onNewEvent }: TopBarProps) {
  const { currentView, currentDate, setView, setCurrentDate, theme, setTheme } =
    useCalendarStore()
  const date = new Date(currentDate)

  const goToday = () => setCurrentDate(startOfToday().toISOString())

  const goPrev = () => {
    const next = currentView === 'month' ? subMonths(date, 1) : subWeeks(date, 1)
    setCurrentDate(next.toISOString())
  }

  const goNext = () => {
    const next = currentView === 'month' ? addMonths(date, 1) : addWeeks(date, 1)
    setCurrentDate(next.toISOString())
  }

  const title = currentView === 'month' ? formatMonthYear(date) : formatWeekTitle(date)

  const toggleTheme = () => {
    if (theme === 'system') setTheme('dark')
    else if (theme === 'dark') setTheme('light')
    else setTheme('system')
  }

  const themeIcon = theme === 'dark' ? '☀️' : theme === 'light' ? '🌙' : '🖥'

  return (
    <header className="topbar">
      <div className="topbar-logo">cala<span>bunga</span></div>
      <button className="topbar-today-btn" onClick={goToday}>Today</button>
      <button className="topbar-nav-btn" onClick={goPrev} aria-label="Previous">‹</button>
      <button className="topbar-nav-btn" onClick={goNext} aria-label="Next">›</button>
      <span className="topbar-title">{title}</span>
      <div className="topbar-view-switcher">
        <button
          className={`topbar-view-btn${currentView === 'month' ? ' active' : ''}`}
          onClick={() => setView('month')}
        >
          Month
        </button>
        <button
          className={`topbar-view-btn${currentView === 'week' ? ' active' : ''}`}
          onClick={() => setView('week')}
        >
          Week
        </button>
      </div>
      <button className="topbar-icon-btn" onClick={toggleTheme} title="Toggle theme">
        {themeIcon}
      </button>
    </header>
  )
}
