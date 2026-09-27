import { FormControlLabel, FormHelperText, MenuItem, Switch, TextField } from '@material-ui/core'

import defaultSettings from './defaultSettings'
import { SettingSectionLabel } from './style'

const groups = [
  {
    title: 'Adaptive startup',
    switches: [
      ['Enabled', 'Enable Flow'],
      ['AdaptiveStartup', 'Adaptive startup buffer'],
    ],
    numbers: [
      ['BootstrapHeadMB', 'Bootstrap head (MB)', 1, 128],
      ['ProbeGraceMs', 'Probe grace (ms)', 0, 10000],
      ['StartupBufferSeconds', 'Startup target (seconds)', 1, 60],
      ['StartupBufferMinMB', 'Minimum startup buffer (MB)', 1, 1024],
      ['StartupBufferMaxMB', 'Maximum startup buffer (MB)', 1, 2048],
      ['StartupSafetyFactorPct', 'Startup safety factor (%)', 100, 300],
    ],
  },
  {
    title: 'Buffering',
    switches: [['AdaptiveReadAhead', 'Adaptive read-ahead']],
    numbers: [
      ['TargetBufferSeconds', 'Target buffer (seconds)', 10, 180],
      ['MaxBufferSeconds', 'Maximum buffer (seconds)', 10, 600],
    ],
  },
  {
    title: 'Mobile sessions and network',
    switches: [],
    numbers: [
      ['WarmSessionTimeoutSec', 'Warm session (seconds)', 30, 1800],
      ['NetworkRetryMinSec', 'Network retry minimum (seconds)', 1, 60],
      ['NetworkRetryMaxSec', 'Network retry maximum (seconds)', 1, 600],
    ],
  },
  {
    title: 'Diagnostics',
    switches: [
      ['MetricsEnabled', 'Collect playback metrics'],
      ['RangeClassification', 'Classify Range requests'],
      ['RangeTraceEnabled', 'Record bounded Range traces'],
      ['DebugFlow', 'Debug Flow logging'],
    ],
    numbers: [],
  },
]

const customFields = [
  ['ConnectionSpeed', 'Connections per second'],
  ['TorrentConnectBoost', 'Torrent connection boost'],
  ['PeerConnectTimeout', 'Peer connection timeout (seconds)'],
  ['PieceTimeout', 'Piece timeout (seconds)'],
  ['RequestQueueTime', 'Request queue time (seconds)'],
  ['MinReconnectTime', 'Minimum reconnect time (seconds)'],
]

export default function FlowSettings({ settings, updateSettings }) {
  // Retain keys added by a future backend when editing a single field.
  const flow = settings.Flow || defaultSettings.Flow
  const change = (key, value) => updateSettings({ Flow: { ...flow, [key]: value } })
  const changeCustom = (key, value) =>
    change('SwarmCustom', {
      ...flow.SwarmCustom,
      [key]: value === '' ? 0 : Number(value),
    })

  return (
    <div style={{ padding: 24, display: 'grid', gap: 24 }}>
      {groups.slice(0, 3).map(group => (
        <section key={group.title} style={{ display: 'grid', gap: 12 }}>
          <SettingSectionLabel>{group.title}</SettingSectionLabel>
          {group.switches.map(([key, label]) => (
            <FormControlLabel
              key={key}
              control={
                <Switch
                  checked={Boolean(flow[key])}
                  onChange={event => change(key, event.target.checked)}
                  color='secondary'
                />
              }
              label={label}
            />
          ))}
          <div
            style={{
              display: 'grid',
              gap: 12,
              gridTemplateColumns: 'repeat(auto-fit, minmax(210px, 1fr))',
            }}
          >
            {group.numbers.map(([key, label, min, max]) => (
              <TextField
                key={key}
                type='number'
                label={label}
                variant='outlined'
                value={flow[key]}
                inputProps={{ min, max }}
                onChange={event => change(key, event.target.value === '' ? 0 : Number(event.target.value))}
              />
            ))}
          </div>
        </section>
      ))}

      <section style={{ display: 'grid', gap: 12 }}>
        <SettingSectionLabel>Swarm profile</SettingSectionLabel>
        <TextField
          select
          label='libtorrent profile'
          variant='outlined'
          value={flow.SwarmProfile}
          onChange={event => change('SwarmProfile', event.target.value)}
        >
          <MenuItem value='legacy'>Legacy (default)</MenuItem>
          <MenuItem value='conservative'>Conservative</MenuItem>
          <MenuItem value='balanced'>Balanced</MenuItem>
          <MenuItem value='aggressive'>Aggressive streaming</MenuItem>
          <MenuItem value='custom'>Custom</MenuItem>
        </TextField>
        <FormHelperText>
          Saving settings restarts the torrent engine and stops active streams. Legacy preserves upstream behavior.
        </FormHelperText>
        {flow.SwarmProfile === 'custom' && (
          <div
            style={{
              display: 'grid',
              gap: 12,
              gridTemplateColumns: 'repeat(auto-fit, minmax(210px, 1fr))',
            }}
          >
            {customFields.map(([key, label]) => (
              <TextField
                key={key}
                type='number'
                label={label}
                variant='outlined'
                value={(flow.SwarmCustom || {})[key] || 0}
                inputProps={{ min: 0 }}
                onChange={event => changeCustom(key, event.target.value)}
                helperText='0 keeps libtorrent default'
              />
            ))}
          </div>
        )}
      </section>

      <section style={{ display: 'grid', gap: 12 }}>
        <SettingSectionLabel>Providers</SettingSectionLabel>
        <FormHelperText>No external provider is configured. Playback uses P2P.</FormHelperText>
      </section>

      <section style={{ display: 'grid', gap: 12 }}>
        <SettingSectionLabel>Diagnostics</SettingSectionLabel>
        {groups[3].switches.map(([key, label]) => (
          <FormControlLabel
            key={key}
            control={
              <Switch
                checked={Boolean(flow[key])}
                onChange={event => change(key, event.target.checked)}
                color='secondary'
              />
            }
            label={label}
          />
        ))}
        <FormHelperText>Open a torrent’s Flow diagnostics from its card while playing.</FormHelperText>
      </section>
    </div>
  )
}
