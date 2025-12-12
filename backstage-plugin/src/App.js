import React, { useState, useEffect, useCallback } from 'react';
import './App.css';

// API base URL - configured via environment or default to localhost
const API_BASE = process.env.REACT_APP_API_URL || 'http://localhost:8081';

function App() {
  const [entries, setEntries] = useState([]);
  const [sites, setSites] = useState([]);
  const [auditLogs, setAuditLogs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [activeTab, setActiveTab] = useState('entries');

  // Form state
  const [formData, setFormData] = useState({
    spiffe_id: '',
    parent_id: 'spiffe://example.org/spire/agent/k8s_sat/demo/default',
    selectors: [{ type: 'k8s:ns', value: '' }],
    site_ids: [],
    description: ''
  });

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const [entriesRes, sitesRes] = await Promise.all([
        fetch(`${API_BASE}/api/v1/entries`),
        fetch(`${API_BASE}/api/v1/sites`)
      ]);

      if (!entriesRes.ok || !sitesRes.ok) {
        throw new Error('Failed to fetch data');
      }

      const entriesData = await entriesRes.json();
      const sitesData = await sitesRes.json();

      setEntries(entriesData.entries || []);
      setSites(sitesData.sites || []);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchAuditLogs = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/v1/audit`);
      if (res.ok) {
        const data = await res.json();
        setAuditLogs(data.entries || []);
      }
    } catch (err) {
      console.error('Failed to fetch audit logs:', err);
    }
  };

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 5000); // Poll every 5 seconds
    return () => clearInterval(interval);
  }, [fetchData]);

  useEffect(() => {
    if (activeTab === 'audit') {
      fetchAuditLogs();
    }
  }, [activeTab]);

  const handleCreateEntry = async (e) => {
    e.preventDefault();
    try {
      const res = await fetch(`${API_BASE}/api/v1/entries`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...formData,
          ttl: 3600
        })
      });

      if (!res.ok) {
        const errData = await res.text();
        throw new Error(errData);
      }

      setShowCreateForm(false);
      setFormData({
        spiffe_id: '',
        parent_id: 'spiffe://example.org/spire/agent/k8s_sat/demo/default',
        selectors: [{ type: 'k8s:ns', value: '' }],
        site_ids: [],
        description: ''
      });
      fetchData();
    } catch (err) {
      alert('Failed to create entry: ' + err.message);
    }
  };

  const handleDeleteEntry = async (id) => {
    if (!window.confirm('Are you sure you want to delete this entry? It will be removed from all sites.')) {
      return;
    }

    try {
      const res = await fetch(`${API_BASE}/api/v1/entries/${id}`, {
        method: 'DELETE'
      });

      if (!res.ok) {
        throw new Error('Failed to delete entry');
      }

      fetchData();
    } catch (err) {
      alert('Failed to delete entry: ' + err.message);
    }
  };

  const addSelector = () => {
    setFormData({
      ...formData,
      selectors: [...formData.selectors, { type: '', value: '' }]
    });
  };

  const updateSelector = (index, field, value) => {
    const newSelectors = [...formData.selectors];
    newSelectors[index][field] = value;
    setFormData({ ...formData, selectors: newSelectors });
  };

  const toggleSite = (siteId) => {
    const newSiteIds = formData.site_ids.includes(siteId)
      ? formData.site_ids.filter(id => id !== siteId)
      : [...formData.site_ids, siteId];
    setFormData({ ...formData, site_ids: newSiteIds });
  };

  const getSyncStatusBadge = (status) => {
    const colors = {
      pending: { bg: '#f59e0b', text: '#fff' },
      synced: { bg: '#10b981', text: '#fff' },
      failed: { bg: '#ef4444', text: '#fff' },
      deleting: { bg: '#8b5cf6', text: '#fff' }
    };
    const color = colors[status] || colors.pending;
    return (
      <span className="status-badge" style={{ backgroundColor: color.bg, color: color.text }}>
        {status.toUpperCase()}
      </span>
    );
  };

  if (loading && entries.length === 0) {
    return (
      <div className="app">
        <header className="header">
          <h1>SPIRE Workload Management</h1>
        </header>
        <main className="main">
          <div className="loading">Loading...</div>
        </main>
      </div>
    );
  }

  return (
    <div className="app">
      <header className="header">
        <div className="header-content">
          <h1>SPIRE Workload Management</h1>
          <span className="user-badge">demo-user</span>
        </div>
      </header>

      <nav className="tabs">
        <button
          className={`tab ${activeTab === 'entries' ? 'active' : ''}`}
          onClick={() => setActiveTab('entries')}
        >
          Workload Entries
        </button>
        <button
          className={`tab ${activeTab === 'sites' ? 'active' : ''}`}
          onClick={() => setActiveTab('sites')}
        >
          Sites
        </button>
        <button
          className={`tab ${activeTab === 'audit' ? 'active' : ''}`}
          onClick={() => setActiveTab('audit')}
        >
          Audit Log
        </button>
      </nav>

      <main className="main">
        {error && (
          <div className="error-banner">
            Error: {error}
            <button onClick={fetchData}>Retry</button>
          </div>
        )}

        {activeTab === 'entries' && (
          <div className="entries-view">
            <div className="toolbar">
              <h2>Workload Entries ({entries.length})</h2>
              <button className="btn-primary" onClick={() => setShowCreateForm(true)}>
                + Create Entry
              </button>
            </div>

            {showCreateForm && (
              <div className="modal-overlay" onClick={() => setShowCreateForm(false)}>
                <div className="modal" onClick={e => e.stopPropagation()}>
                  <h3>Create Workload Entry</h3>
                  <form onSubmit={handleCreateEntry}>
                    <div className="form-group">
                      <label>SPIFFE ID</label>
                      <input
                        type="text"
                        placeholder="spiffe://example.org/workload/my-service"
                        value={formData.spiffe_id}
                        onChange={e => setFormData({ ...formData, spiffe_id: e.target.value })}
                        required
                      />
                    </div>

                    <div className="form-group">
                      <label>Parent ID</label>
                      <input
                        type="text"
                        value={formData.parent_id}
                        onChange={e => setFormData({ ...formData, parent_id: e.target.value })}
                        required
                      />
                    </div>

                    <div className="form-group">
                      <label>Selectors</label>
                      {formData.selectors.map((sel, idx) => (
                        <div key={idx} className="selector-row">
                          <select
                            value={sel.type}
                            onChange={e => updateSelector(idx, 'type', e.target.value)}
                          >
                            <option value="">Select type...</option>
                            <option value="k8s:ns">k8s:ns (namespace)</option>
                            <option value="k8s:sa">k8s:sa (service account)</option>
                            <option value="k8s:pod-label">k8s:pod-label</option>
                          </select>
                          <input
                            type="text"
                            placeholder="value"
                            value={sel.value}
                            onChange={e => updateSelector(idx, 'value', e.target.value)}
                          />
                        </div>
                      ))}
                      <button type="button" className="btn-secondary" onClick={addSelector}>
                        + Add Selector
                      </button>
                    </div>

                    <div className="form-group">
                      <label>Target Sites</label>
                      <div className="site-checkboxes">
                        {sites.map(site => (
                          <label key={site.ID} className="checkbox-label">
                            <input
                              type="checkbox"
                              checked={formData.site_ids.includes(site.ID)}
                              onChange={() => toggleSite(site.ID)}
                            />
                            {site.Name} ({site.Region})
                          </label>
                        ))}
                      </div>
                    </div>

                    <div className="form-group">
                      <label>Description</label>
                      <textarea
                        value={formData.description}
                        onChange={e => setFormData({ ...formData, description: e.target.value })}
                        placeholder="Optional description..."
                      />
                    </div>

                    <div className="form-actions">
                      <button type="button" className="btn-secondary" onClick={() => setShowCreateForm(false)}>
                        Cancel
                      </button>
                      <button type="submit" className="btn-primary">
                        Create Entry
                      </button>
                    </div>
                  </form>
                </div>
              </div>
            )}

            <div className="entries-table">
              <table>
                <thead>
                  <tr>
                    <th>SPIFFE ID</th>
                    <th>Selectors</th>
                    <th>Site Status</th>
                    <th>Created By</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {entries.length === 0 ? (
                    <tr>
                      <td colSpan="5" className="empty">No workload entries found</td>
                    </tr>
                  ) : (
                    entries.map(entry => (
                      <tr key={entry.ID}>
                        <td className="spiffe-id">{entry.SpiffeID}</td>
                        <td>
                          {entry.Selectors?.map((sel, idx) => (
                            <span key={idx} className="selector-tag">
                              {sel.Type}:{sel.Value}
                            </span>
                          ))}
                        </td>
                        <td>
                          <div className="site-statuses">
                            {entry.SiteStatuses?.map(st => (
                              <div key={st.SiteID} className="site-status">
                                <span className="site-name">{st.SiteName}</span>
                                {getSyncStatusBadge(st.Status)}
                                {st.SyncError && (
                                  <span className="error-tooltip" title={st.SyncError}>!</span>
                                )}
                              </div>
                            ))}
                          </div>
                        </td>
                        <td>{entry.CreatedBy}</td>
                        <td>
                          <button
                            className="btn-danger"
                            onClick={() => handleDeleteEntry(entry.ID)}
                          >
                            Delete
                          </button>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {activeTab === 'sites' && (
          <div className="sites-view">
            <h2>Configured Sites ({sites.length})</h2>
            <div className="sites-grid">
              {sites.map(site => (
                <div key={site.ID} className="site-card">
                  <h3>{site.Name}</h3>
                  <div className="site-details">
                    <div><strong>ID:</strong> {site.ID}</div>
                    <div><strong>Region:</strong> {site.Region}</div>
                    <div><strong>Trust Domain:</strong> {site.TrustDomain}</div>
                    <div><strong>Status:</strong> {getSyncStatusBadge(site.Status)}</div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {activeTab === 'audit' && (
          <div className="audit-view">
            <h2>Audit Log</h2>
            <div className="audit-table">
              <table>
                <thead>
                  <tr>
                    <th>Timestamp</th>
                    <th>Actor</th>
                    <th>Action</th>
                    <th>Resource</th>
                    <th>Details</th>
                  </tr>
                </thead>
                <tbody>
                  {auditLogs.length === 0 ? (
                    <tr>
                      <td colSpan="5" className="empty">No audit logs found</td>
                    </tr>
                  ) : (
                    auditLogs.map(log => (
                      <tr key={log.ID}>
                        <td>{new Date(log.Timestamp?.seconds * 1000).toLocaleString()}</td>
                        <td>{log.Actor}</td>
                        <td><span className={`action-badge ${log.Action}`}>{log.Action}</span></td>
                        <td>{log.ResourceType}/{log.ResourceID?.substring(0, 8)}...</td>
                        <td className="details">{log.Details}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </main>
    </div>
  );
}

export default App;
