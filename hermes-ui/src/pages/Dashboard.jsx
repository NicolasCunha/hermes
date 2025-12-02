import { useState, useEffect } from 'react'
import { serviceService } from '../services/api'
import './Dashboard.css'

const Dashboard = () => {
  const [services, setServices] = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadServices()
  }, [])

  const loadServices = async () => {
    try {
      const data = await serviceService.getAll()
      setServices(data.services || [])
    } catch {
      // Silently fail - dashboard will show empty state
    } finally {
      setLoading(false)
    }
  }

  if (loading) {
    return <div className="loading">Loading dashboard...</div>
  }

  const healthyCount = services.filter(s => s.status === 'healthy').length
  const unhealthyCount = services.filter(s => s.status === 'unhealthy').length

  return (
    <div className="dashboard">
      <h1>Dashboard</h1>
      
      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-icon">🔌</div>
          <div className="stat-content">
            <h3>Total Services</h3>
            <p className="stat-value">{services.length}</p>
          </div>
        </div>
        
        <div className="stat-card healthy">
          <div className="stat-icon">✅</div>
          <div className="stat-content">
            <h3>Healthy Services</h3>
            <p className="stat-value">{healthyCount}</p>
          </div>
        </div>
        
        <div className="stat-card unhealthy">
          <div className="stat-icon">⚠️</div>
          <div className="stat-content">
            <h3>Unhealthy Services</h3>
            <p className="stat-value">{unhealthyCount}</p>
          </div>
        </div>
      </div>

      {services.length > 0 && (
        <div className="services-section">
          <h2>Registered Services</h2>
          <div className="services-list">
            {services.map((service) => (
              <div key={service.id} className="service-card">
                <div className="service-header">
                  <h3>{service.name}</h3>
                  <span className={`health-badge ${service.status}`}>
                    {service.status === 'healthy' ? '✓' : '✗'} {service.status}
                  </span>
                </div>
                <div className="service-details">
                  <p><strong>URL:</strong> {service.protocol}://{service.host}:{service.port}</p>
                  <p><strong>Health Check:</strong> {service.health_check_path}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {services.length === 0 && (
        <div className="welcome-section">
          <h2>Welcome to Hermes</h2>
          <p>
            Hermes is an API Gateway that provides centralized authentication,
            service registry, and routing capabilities for your microservices architecture.
          </p>
          <div className="quick-links">
            <a href="/services" className="quick-link">
              <span>🔌</span>
              <span>Manage Services</span>
            </a>
            <a href="/users" className="quick-link">
              <span>👥</span>
              <span>Manage Users</span>
            </a>
          </div>
        </div>
      )}
    </div>
  )
}

export default Dashboard
