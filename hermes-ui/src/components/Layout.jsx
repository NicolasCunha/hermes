import { Outlet, Link, useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import './Layout.css'

const Layout = () => {
  const { logout, user } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  const isActive = (path) => {
    return location.pathname === path ? 'active' : ''
  }

  // Check if user is admin by decoding token
  const isAdmin = () => {
    const token = sessionStorage.getItem('hermes_token')
    if (!token) return false
    
    try {
      const payload = token.split('.')[1]
      const decoded = JSON.parse(atob(payload))
      return decoded.roles?.includes('admin') || false
    } catch {
      return false
    }
  }

  const userIsAdmin = isAdmin()

  return (
    <div className="layout">
      <nav className="sidebar">
        <div className="sidebar-header">
          <h2>Hermes</h2>
          <p className="subtitle">API Gateway</p>
        </div>
        
        <ul className="nav-menu">
          <li>
            <Link to="/dashboard" className={isActive('/dashboard')}>
              <span className="icon">📊</span>
              Dashboard
            </Link>
          </li>
          {userIsAdmin && (
            <>
              <li>
                <Link to="/services" className={isActive('/services')}>
                  <span className="icon">🔌</span>
                  Services
                </Link>
              </li>
              <li>
                <Link to="/users" className={isActive('/users')}>
                  <span className="icon">👥</span>
                  Users
                </Link>
              </li>
            </>
          )}
          <li>
            <Link to="/profile" className={isActive('/profile')}>
              <span className="icon">⚙️</span>
              Profile
            </Link>
          </li>
        </ul>

        <div className="sidebar-footer">
          <div className="user-info">
            <span className="icon">👤</span>
            <span className="username">{user?.subject || 'Admin'}</span>
          </div>
          <button onClick={handleLogout} className="logout-btn">
            Logout
          </button>
        </div>
      </nav>

      <main className="main-content">
        <Outlet />
      </main>
    </div>
  )
}

export default Layout
