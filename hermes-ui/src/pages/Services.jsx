import { useState, useEffect } from 'react'
import { serviceService } from '../services/api'
import Modal from '../components/Modal'
import './Services.css'

const Services = () => {
  const [services, setServices] = useState([])
  const [loading, setLoading] = useState(true)
  const [showForm, setShowForm] = useState(false)
  const [formData, setFormData] = useState({
    name: '',
    base_url: '',
    health_check_path: '/health'
  })
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [deleteModal, setDeleteModal] = useState({ isOpen: false, service: null })
  const [healthLogs, setHealthLogs] = useState({ isOpen: false, service: null, logs: [], loading: false })
  const [expandedLogId, setExpandedLogId] = useState(null)

  useEffect(() => {
    loadServices()
  }, [])

  const loadServices = async () => {
    try {
      const data = await serviceService.getAll()
      setServices(data.services || [])
    } catch {
      setError('Failed to load services')
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

  const handleSubmit = async (e) => {
    e.preventDefault()
    setError('')
    setSubmitting(true)

    try {
      // Parse base_url to extract host and port
      const url = new URL(formData.base_url)
      const payload = {
        name: formData.name,
        host: url.hostname,
        port: parseInt(url.port) || (url.protocol === 'https:' ? 443 : 80),
        health_check_path: formData.health_check_path,
        protocol: url.protocol.replace(':', '')
      }
      
      await serviceService.register(payload)
      setShowForm(false)
      setFormData({ name: '', base_url: '', health_check_path: '/health' })
      await loadServices()
    } catch (error) {
      setError(error.response?.data?.error || 'Failed to register service')
    } finally {
      setSubmitting(false)
    }
  }

  const confirmDelete = async () => {
    if (!deleteModal.service) return

    try {
      await serviceService.delete(deleteModal.service.id)
      setDeleteModal({ isOpen: false, service: null })
      await loadServices()
    } catch (error) {
      setError('Failed to delete service: ' + (error.response?.data?.error || error.message))
      setDeleteModal({ isOpen: false, service: null })
    }
  }

  const viewHealthLogs = async (service) => {
    setHealthLogs({ isOpen: true, service, logs: [], loading: true })
    
    try {
      const response = await serviceService.getHealthLogs(service.id, 20)
      setHealthLogs(prev => ({ ...prev, logs: response.logs || [], loading: false }))
    } catch (error) {
      setError('Failed to load health logs')
      setHealthLogs(prev => ({ ...prev, loading: false }))
    }
  }

  const closeHealthLogs = () => {
    setHealthLogs({ isOpen: false, service: null, logs: [], loading: false })
    setExpandedLogId(null)
  }

  const toggleLogExpand = (logId) => {
    setExpandedLogId(expandedLogId === logId ? null : logId)
  }

  const getStatusBadge = (status) => {
    if (status === 'healthy') {
      return <span className="badge badge-healthy">✓ Healthy</span>
    }
    return <span className="badge badge-unhealthy">⚠ Unhealthy</span>
  }

  if (loading) {
    return <div className="loading">Loading services...</div>
  }

  return (
    <div className="services-page">
      <div className="page-header">
        <h1>Services</h1>
        <button onClick={() => setShowForm(!showForm)} className="btn-primary">
          {showForm ? 'Cancel' : '+ Register Service'}
        </button>
      </div>

      {showForm && (
        <div className="service-form">
          <h2>Register New Service</h2>
          {error && <div className="error-message">{error}</div>}
          
          <form onSubmit={handleSubmit}>
            <div className="form-row">
              <div className="form-group">
                <label>Service Name *</label>
                <input
                  type="text"
                  name="name"
                  value={formData.name}
                  onChange={handleInputChange}
                  required
                  placeholder="e.g., user-service"
                />
              </div>
              
              <div className="form-group">
                <label>Base URL *</label>
                <input
                  type="url"
                  name="base_url"
                  value={formData.base_url}
                  onChange={handleInputChange}
                  required
                  placeholder="e.g., http://localhost:8081"
                />
              </div>
              
              <div className="form-group">
                <label>Health Check Path</label>
                <input
                  type="text"
                  name="health_check_path"
                  value={formData.health_check_path}
                  onChange={handleInputChange}
                  placeholder="/health"
                />
              </div>
            </div>
            
            <button type="submit" disabled={submitting} className="btn-submit">
              {submitting ? 'Registering...' : 'Register Service'}
            </button>
          </form>
        </div>
      )}

      {services.length === 0 ? (
        <div className="empty-state">
          <p>No services registered yet.</p>
          <button onClick={() => setShowForm(true)} className="btn-primary">
            Register First Service
          </button>
        </div>
      ) : (
        <div className="services-table">
          <table>
            <thead>
              <tr>
                <th>Service Name</th>
                <th>Status</th>
                <th>Base URL</th>
                <th>Health Check Path</th>
                <th>Last Check</th>
                <th>Failures</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {services.map((service) => (
                <tr key={service.id}>
                  <td>{service.name}</td>
                  <td>{getStatusBadge(service.status)}</td>
                  <td>
                    {service.protocol}://{service.host}:{service.port}
                  </td>
                  <td>{service.health_check_path}</td>
                  <td>
                    {service.last_checked_at 
                      ? new Date(service.last_checked_at).toLocaleString()
                      : 'Never'
                    }
                  </td>
                  <td>
                    {service.failure_count || 0}
                  </td>
                  <td>
                    <div className="action-buttons">
                      <button 
                        onClick={() => viewHealthLogs(service)}
                        className="btn-edit-small"
                      >
                        View Logs
                      </button>
                      <button 
                        onClick={() => setDeleteModal({ isOpen: true, service })}
                        className="btn-delete-small"
                      >
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Modal
        isOpen={deleteModal.isOpen}
        onClose={() => setDeleteModal({ isOpen: false, service: null })}
        onConfirm={confirmDelete}
        title="Delete Service"
        message={`Are you sure you want to delete the service "${deleteModal.service?.name}"? This action cannot be undone.`}
        confirmText="Delete"
        cancelText="Cancel"
      />

      {healthLogs.isOpen && (
        <div className="modal-overlay" onClick={closeHealthLogs}>
          <div className="health-logs-modal" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2>Health Check Logs - {healthLogs.service?.name}</h2>
              <button onClick={closeHealthLogs} className="close-button">×</button>
            </div>
            <div className="modal-body">
              {healthLogs.loading ? (
                <div className="loading">Loading logs...</div>
              ) : healthLogs.logs.length === 0 ? (
                <div className="empty-state">No health check logs available yet.</div>
              ) : (
                <table className="logs-table">
                  <thead>
                    <tr>
                      <th>Time</th>
                      <th>Status</th>
                      <th>Response Time</th>
                      <th>Error</th>
                      <th>Details</th>
                    </tr>
                  </thead>
                  <tbody>
                    {healthLogs.logs.map((log) => (
                      <>
                        <tr key={log.id}>
                          <td>{new Date(log.checked_at).toLocaleString()}</td>
                          <td>
                            <span className={`badge badge-${log.status}`}>
                              {log.status}
                            </span>
                          </td>
                          <td>{log.response_time_ms}ms</td>
                          <td className="error-cell">{log.error_message || '-'}</td>
                          <td>
                            {log.response_body && (
                              <button 
                                onClick={() => toggleLogExpand(log.id)}
                                className="btn-expand"
                              >
                                {expandedLogId === log.id ? '▼ Hide' : '▶ View Response'}
                              </button>
                            )}
                          </td>
                        </tr>
                        {expandedLogId === log.id && log.response_body && (
                          <tr key={`${log.id}-detail`} className="log-detail-row">
                            <td colSpan="5">
                              <div className="response-body">
                                <strong>Response Body:</strong>
                                <pre>{log.response_body}</pre>
                              </div>
                            </td>
                          </tr>
                        )}
                      </>
                    ))}
                  </tbody>
                </table>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default Services
