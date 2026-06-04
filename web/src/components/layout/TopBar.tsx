import { addMonths, addWeeks, subMonths, subWeeks, startOfToday } from 'date-fns'
import { MdChevronLeft, MdChevronRight } from 'react-icons/md'
import { useCalendarStore } from '../../store/calendarStore'
import { formatMonthYear, formatWeekTitle } from '../../lib/dateUtils'

interface TopBarProps {
  onNewEvent: () => void
}

export default function TopBar({ onNewEvent: _onNewEvent }: TopBarProps) {
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

  return (
    <header className="topbar">
      <div className="topbar-logo">cala<span>bunga</span></div>
      <button className="topbar-today-btn" onClick={goToday}>Today</button>
      <button className="topbar-nav-btn" onClick={goPrev} aria-label="Previous">
        <MdChevronLeft size={22} />
      </button>
      <button className="topbar-nav-btn" onClick={goNext} aria-label="Next">
        <MdChevronRight size={22} />
      </button>
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
      <div className="topbar-view-switcher">
        <button
          className={`topbar-view-btn${theme === 'light' ? ' active' : ''}`}
          onClick={() => setTheme('light')}
        >
          Light
        </button>
        <button
          className={`topbar-view-btn${theme === 'dark' ? ' active' : ''}`}
          onClick={() => setTheme('dark')}
        >
          Dark
        </button>
        <button
          className={`topbar-view-btn${theme === 'system' ? ' active' : ''}`}
          onClick={() => setTheme('system')}
        >
          System
        </button>
      </div>
    </header>
  )
}
