import { useState, useEffect } from 'react'
import { userService } from '../services/api'
import Modal from '../components/Modal'
import './Users.css'

const Users = () => {
  const [users, setUsers] = useState([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)
  const [formData, setFormData] = useState({
    subject: '',
    password: '',
    roles: []
  })
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [editingUserId, setEditingUserId] = useState(null)
  const [editingRoles, setEditingRoles] = useState([])
  const [editError, setEditError] = useState('')
  const [deleteModal, setDeleteModal] = useState({ isOpen: false, user: null })

  const getCurrentUserId = () => {
    const token = sessionStorage.getItem('hermes_token')
    if (!token) return null
    try {
      const payload = token.split('.')[1]
      const decoded = JSON.parse(atob(payload))
      return decoded.user_id
    } catch {
      return null
    }
  }

  useEffect(() => {
    loadUsers()
  }, [])

  const loadUsers = async () => {
    try {
      const data = await userService.getAll()
      // Handle both array and object responses
      setUsers(Array.isArray(data) ? data : (data.users || []))
    } catch {
      setError('Failed to load users')
    } finally {
      setLoading(false)
    }
  }

  const handleInputChange = (e) => {
    setFormData({
      ...formData,
      [e.target.name]: e.target.value
    })
  }

  const handleRoleToggle = (role) => {
    const currentRoles = formData.roles || []
    if (currentRoles.includes(role)) {
      setFormData({
        ...formData,
        roles: currentRoles.filter(r => r !== role)
      })
    } else {
      setFormData({
        ...formData,
        roles: [...currentRoles, role]
      })
    }
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setSubmitting(true)

    try {
      await userService.create(formData)
      setShowForm(false)
      setFormData({ subject: '', password: '', roles: [] })
      await loadUsers()
    } catch (error) {
      const errorMsg = error.response?.data?.error || error.message || 'Failed to create user'
      setError(errorMsg)
    } finally {
      setSubmitting(false)
    }
  }

  const confirmDelete = async () => {
    if (!deleteModal.user) return

    try {
      await userService.delete(deleteModal.user.id)
      setDeleteModal({ isOpen: false, user: null })
      await loadUsers()
    } catch (error) {
      setError('Failed to delete user: ' + (error.response?.data?.error || error.message))
      setDeleteModal({ isOpen: false, user: null })
    }
  }

  const startEditRoles = (user) => {
    setEditingUserId(user.id)
    setEditingRoles(user.roles || [])
    setEditError('')
  }

  const cancelEditRoles = () => {
    setEditingUserId(null)
    setEditingRoles([])
    setEditError('')
  }

  const toggleEditRole = (role) => {
    if (editingRoles.includes(role)) {
      setEditingRoles(editingRoles.filter(r => r !== role))
    } else {
      setEditingRoles([...editingRoles, role])
    }
  }

  const saveRoles = async (userId) => {
    setEditError('')
    try {
      await userService.updateRoles(userId, editingRoles)
      setEditingUserId(null)
      setEditingRoles([])
      await loadUsers()
    } catch (error) {
      const errorMsg = error.response?.data?.error || error.message || 'Failed to update roles'
      setEditError(errorMsg)
    }
  }

  if (loading) {
    return <div className="loading">Loading users...</div>
  }

  return (
    <div className="users-page">
      <div className="page-header">
        <h1>Users</h1>
        <button onClick={() => setShowForm(!showForm)} className="btn-primary">
          {showForm ? 'Cancel' : '+ Create User'}
        </button>
      </div>

      {showForm && (
        <div className="user-form">
          <h2>Create New User</h2>
          {error && <div className="error-message">{error}</div>}
          
          <form onSubmit={handleSubmit}>
            <div className="form-row">
              <div className="form-group">
                <label>Username *</label>
                <input
                  type="text"
                  name="subject"
                  value={formData.subject}
                  onChange={handleInputChange}
                  required
                  placeholder="e.g., john@example.com"
                />
              </div>
              
              <div className="form-group">
                <label>Password * (min 8 characters)</label>
                <input
                  type="password"
                  name="password"
                  value={formData.password}
                  onChange={handleInputChange}
                  required
                  minLength="8"
                  placeholder="Enter password"
                />
              </div>
            </div>

            <div className="form-group">
              <label>Roles</label>
              <div className="checkbox-group">
                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={formData.roles.includes('admin')}
                    onChange={() => handleRoleToggle('admin')}
                  />
                  <span>Admin</span>
                </label>
                <label className="checkbox-label">
                  <input
                    type="checkbox"
                    checked={formData.roles.includes('member')}
                    onChange={() => handleRoleToggle('member')}
                  />
                  <span>Member</span>
                </label>
              </div>
            </div>
            
            <button type="submit" disabled={submitting} className="btn-submit">
              {submitting ? 'Creating...' : 'Create User'}
            </button>
          </form>
        </div>
      )}

      {editError && <div className="error-message">{editError}</div>}

      <div className="users-list">
        {users.length === 0 ? (
          <div className="empty-state">
            <p>No users found.</p>
          </div>
        ) : (
          <div className="users-table">
            <table>
              <thead>
                <tr>
                  <th>Username</th>
                  <th>Roles</th>
                  <th>Created At</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {users.map((user) => (
                  <tr key={user.id}>
                    <td>{user.subject}</td>
                    <td>
                      {editingUserId === user.id ? (
                        <div className="role-edit">
                          <label className="checkbox-label-inline">
                            <input
                              type="checkbox"
                              checked={editingRoles.includes('admin')}
                              onChange={() => toggleEditRole('admin')}
                            />
                            Admin
                          </label>
                          <label className="checkbox-label-inline">
                            <input
                              type="checkbox"
                              checked={editingRoles.includes('member')}
                              onChange={() => toggleEditRole('member')}
                            />
                            Member
                          </label>
                        </div>
                      ) : (
                        <div className="roles-badges">
                          {user.roles && user.roles.length > 0 ? (
                            user.roles.map((role) => (
                              <span key={role} className="role-badge">
                                {role}
                              </span>
                            ))
                          ) : (
                            <span className="no-roles">No roles</span>
                          )}
                        </div>
                      )}
                    </td>
                    <td>{new Date(user.created_at).toLocaleDateString()}</td>
                    <td>
                      {editingUserId === user.id ? (
                        <div className="action-buttons">
                          <button
                            onClick={() => saveRoles(user.id)}
                            className="btn-save-small"
                          >
                            Save
                          </button>
                          <button
                            onClick={cancelEditRoles}
                            className="btn-cancel-small"
                          >
                            Cancel
                          </button>
                        </div>
                      ) : (
                        <div className="action-buttons">
                          {getCurrentUserId() !== user.id && (
                            <>
                              <button
                                onClick={() => startEditRoles(user)}
                                className="btn-edit-small"
                              >
                                Edit Roles
                              </button>
                              <button
                                onClick={() => setDeleteModal({ isOpen: true, user })}
                                className="btn-delete-small"
                              >
                                Delete
                              </button>
                            </>
                          )}
                          {getCurrentUserId() === user.id && (
                            <span className="current-user-label">Current User</span>
                          )}
                        </div>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <Modal
        isOpen={deleteModal.isOpen}
        onClose={() => setDeleteModal({ isOpen: false, user: null })}
        onConfirm={confirmDelete}
        title="Delete User"
        message={`Are you sure you want to delete user "${deleteModal.user?.subject}"? This action cannot be undone.`}
        confirmText="Delete"
        cancelText="Cancel"
      />
    </div>
  )
}

export default Users
