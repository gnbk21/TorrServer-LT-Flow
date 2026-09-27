import axios from 'axios'
import { Button, Dialog, DialogActions, DialogContent, DialogTitle } from '@material-ui/core'
import { useEffect, useState } from 'react'
import { flowStatusHost } from 'utils/Hosts'

const mb = bytes => `${(Number(bytes || 0) / 1048576).toFixed(0)} MB`
const mbps = bytesPerSecond => `${((Number(bytesPerSecond || 0) * 8) / 1000000).toFixed(1)} Mbps`
const seconds = value => `${Number(value || 0).toFixed(1)} s`

const Metric = ({ label, value }) => (
  <div
    style={{
      display: 'flex',
      justifyContent: 'space-between',
      gap: 16,
      borderBottom: '1px solid #8884',
      padding: '5px 0',
    }}
  >
    <span>{label}</span>
    <strong>{value}</strong>
  </div>
)

export default function FlowDiagnostics({ hash, onClose }) {
  const [status, setStatus] = useState(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let active = true
    const refresh = () => {
      axios
        .get(flowStatusHost(hash))
        .then(({ data }) => {
          if (active) {
            setStatus(data)
            setError('')
          }
        })
        .catch(err => {
          if (active) setError(err.response?.data?.error || err.message)
        })
    }
    refresh()
    const interval = window.setInterval(refresh, 2000)
    return () => {
      active = false
      window.clearInterval(interval)
    }
  }, [hash])

  return (
    <Dialog open onClose={onClose} fullWidth maxWidth='sm'>
      <DialogTitle>Flow diagnostics</DialogTitle>
      <DialogContent dividers>
        {error && <p role='alert'>{error}</p>}
        {!status && !error && <p>Loading…</p>}
        {status && (
          <>
            <Metric label='Startup state' value={status.startup?.state || 'UNKNOWN'} />
            <Metric label='Local address' value={status.network?.state || 'UNKNOWN'} />
            <Metric label='Tracker connectivity' value={status.network?.connectivity || 'INTERNET_WAIT'} />
            <Metric label='Startup time to first byte' value={`${status.startup?.time_to_first_byte_ms || 0} ms`} />
            {(status.sessions || []).length === 0 && <p>No playback session yet.</p>}
            {(status.sessions || []).map(session => (
              <section key={`${session.group}-${session.file_index}`} style={{ marginTop: 20 }}>
                <h3 style={{ marginBottom: 6 }}>
                  File {session.file_index + 1} · {session.state}
                </h3>
                <Metric
                  label='Media bitrate'
                  value={
                    session.estimated_media_bitrate > 0 ? mbps(Number(session.estimated_media_bitrate) / 8) : 'Unknown'
                  }
                />
                <Metric label='Download' value={mbps(session.download_rate)} />
                <Metric
                  label='Sustainability'
                  value={
                    session.playback_consumption_rate > 0
                      ? `${Number(session.sustainability_ratio || 0).toFixed(2)}×`
                      : 'Unknown'
                  }
                />
                <Metric label='Buffer ahead' value={seconds(session.buffer_ahead_seconds)} />
                <Metric label='Buffer risk' value={session.buffer_warning ? 'Warning' : 'Normal'} />
                <Metric label='Cache' value={`${mb(session.cache_used)} / ${mb(session.cache_size)}`} />
                <Metric label='Connected peers' value={session.connected_peers ?? 0} />
                <Metric label='Piece waits' value={session.piece_wait_count ?? 0} />
                <Metric label='Warm session' value={session.state === 'WARM_IDLE' ? 'Active' : 'No'} />
                <Metric label='Last seek recovery' value={`${session.seek_recovery_ms || 0} ms`} />
                <Metric label='Last HTTP time to first byte' value={`${session.last_ttfb_ms || 0} ms`} />
                <Metric label='Source' value='P2P' />
              </section>
            ))}
            {(status.trackers || []).length > 0 && (
              <section style={{ marginTop: 20 }}>
                <h3>Trackers</h3>
                {status.trackers.map(tracker => (
                  <Metric
                    key={tracker.id}
                    label={`${tracker.protocol}://${tracker.host}`}
                    value={`${tracker.status}${tracker.error ? ` · ${tracker.error}` : ` · ${tracker.peers} peers`}`}
                  />
                ))}
              </section>
            )}
          </>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} color='secondary'>
          Close
        </Button>
      </DialogActions>
    </Dialog>
  )
}
