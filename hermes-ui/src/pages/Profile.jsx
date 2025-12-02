import { useState } from 'react'
import { userService, authService } from '../services/api'
import './Profile.css'

const Profile = () => {
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [loading, setLoading] = useState(false)

  const getUserIdFromToken = () => {
    const token = authService.getToken()
    if (!token) return null
    
    try {
      // Decode JWT token (basic decode, no validation)
      const payload = token.split('.')[1]
      const decoded = JSON.parse(atob(payload))
      return decoded.user_id
    } catch {
      return null
    }
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setSuccess('')

    if (newPassword !== confirmPassword) {
      setError('New passwords do not match')
      return
    }

    if (newPassword.length < 8) {
      setError('Password must be at least 8 characters')
      return
    }

    const userId = getUserIdFromToken()
    if (!userId) {
      setError('Unable to determine user ID')
      return
    }

    setLoading(true)

    try {
      await userService.changePassword(userId, oldPassword, newPassword)
      setSuccess('Password changed successfully')
      setOldPassword('')
      setNewPassword('')
      setConfirmPassword('')
    } catch (error) {
      setError(error.response?.data?.error || 'Failed to change password')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="profile-page">
      <h1>Profile Settings</h1>

      <div className="profile-card">
        <h2>Change Password</h2>
        
        {error && <div className="error-message">{error}</div>}
        {success && <div className="success-message">{success}</div>}

        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label htmlFor="oldPassword">Current Password</label>
            <input
              id="oldPassword"
              type="password"
              value={oldPassword}
              onChange={(e) => setOldPassword(e.target.value)}
              required
              placeholder="Enter current password"
            />
          </div>

          <div className="form-group">
            <label htmlFor="newPassword">New Password</label>
            <input
              id="newPassword"
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              required
              minLength="8"
              placeholder="Enter new password (min 8 characters)"
            />
          </div>

          <div className="form-group">
            <label htmlFor="confirmPassword">Confirm New Password</label>
            <input
              id="confirmPassword"
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              required
              minLength="8"
              placeholder="Confirm new password"
            />
          </div>

          <button type="submit" disabled={loading} className="btn-submit">
            {loading ? 'Changing Password...' : 'Change Password'}
          </button>
        </form>
      </div>

      <div className="profile-info">
        <h2>About Hermes</h2>
        <p>
          Hermes is an API Gateway that provides centralized authentication,
          service registry, and routing capabilities. It integrates with Aegis
          for authentication and authorization.
        </p>
      </div>
    </div>
  )
}

export default Profile
