param(
    [ValidateRange(1, 65535)][int]$Port = 8090,
    [string]$ServerHost = '127.0.0.1',
    [System.Management.Automation.PSCredential]$Credential
)

$ErrorActionPreference = 'Stop'
if ($Credential -and $ServerHost -notin @('127.0.0.1', 'localhost', '::1', '[::1]')) {
    throw 'Use loopback when passing HTTP credentials to the tray.'
}
if ($ServerHost -eq '::1') { $ServerHost = '[::1]' }
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
[System.Windows.Forms.Application]::EnableVisualStyles()

$script:baseUri = "http://${ServerHost}:${Port}"
$script:Credential = $Credential
$script:context = [System.Windows.Forms.ApplicationContext]::new()
$script:menu = [System.Windows.Forms.ContextMenuStrip]::new()
$script:menu.ShowImageMargin = $false
$script:stateItem = $script:menu.Items.Add('Server: connecting')
$script:titleItem = $script:menu.Items.Add('Active torrent: none')
$script:speedItem = $script:menu.Items.Add('Download: 0 MiB/s')
$script:bufferItem = $script:menu.Items.Add('Buffer: 0 s')
$script:peersItem = $script:menu.Items.Add('Peers: 0')
foreach ($item in @($script:stateItem, $script:titleItem, $script:speedItem, $script:bufferItem, $script:peersItem)) {
    $item.Enabled = $false
}
[void]$script:menu.Items.Add([System.Windows.Forms.ToolStripSeparator]::new())
$script:openItem = $script:menu.Items.Add('Open UI')
$script:restartItem = $script:menu.Items.Add('Restart Service')
$script:pauseItem = $script:menu.Items.Add('Pause')
$script:resumeItem = $script:menu.Items.Add('Resume')
[void]$script:menu.Items.Add([System.Windows.Forms.ToolStripSeparator]::new())
$script:exitItem = $script:menu.Items.Add('Exit Tray')

$script:icon = [System.Windows.Forms.NotifyIcon]::new()
$script:icon.Icon = [System.Drawing.SystemIcons]::Application
$script:icon.Text = 'TorrServer-Flow'
$script:icon.ContextMenuStrip = $script:menu
$script:icon.Visible = $true
$script:timer = [System.Windows.Forms.Timer]::new()
$script:timer.Interval = 5000

function Invoke-FlowRequest {
    param([string]$Path, [string]$Action = '')
    $request = @{
        Uri        = "$script:baseUri$Path"
        TimeoutSec = 2
        ErrorAction = 'Stop'
    }
    if ($script:Credential) {
        $request.Credential = $script:Credential
    }
    if ($Action) {
        $request.Method = 'Post'
        $request.ContentType = 'application/json'
        $request.Body = @{ action = $Action } | ConvertTo-Json -Compress
    }
    return Invoke-RestMethod @request
}

function Update-FlowTray {
    try {
        $status = Invoke-FlowRequest '/flow/tray'
        $state = [string]$status.server_state
        $title = [string]$status.active_torrent
        if ([string]::IsNullOrWhiteSpace($title)) { $title = 'none' }
        if ($title.Length -gt 45) { $title = $title.Substring(0, 42) + '...' }
        $script:stateItem.Text = "Server: $state"
        $script:titleItem.Text = "Active torrent: $title"
        $script:speedItem.Text = ('Download: {0:N1} MiB/s' -f ([double]$status.download_rate / 1MB))
        $script:bufferItem.Text = ('Buffer: {0:N0} s' -f [double]$status.buffer_seconds)
        $script:peersItem.Text = "Peers: $($status.peer_count)"
        $script:icon.Text = "TorrServer-Flow - $state"
        $script:pauseItem.Enabled = $state -eq 'RUNNING'
        $script:resumeItem.Enabled = $state -eq 'PAUSED'
    } catch {
        $script:stateItem.Text = 'Server: unavailable'
        $script:titleItem.Text = 'Active torrent: none'
        $script:speedItem.Text = 'Download: 0 MiB/s'
        $script:bufferItem.Text = 'Buffer: 0 s'
        $script:peersItem.Text = 'Peers: 0'
        $script:icon.Text = 'TorrServer-Flow - unavailable'
        $script:pauseItem.Enabled = $false
        $script:resumeItem.Enabled = $false
    }
}

function Invoke-FlowAction {
    param([string]$Action)
    try {
        [void](Invoke-FlowRequest '/flow/control' $Action)
        Update-FlowTray
    } catch {
        [void][System.Windows.Forms.MessageBox]::Show($_.Exception.Message, 'TorrServer-Flow')
    }
}

$script:timer.Add_Tick({ Update-FlowTray })
$script:openItem.Add_Click({ Start-Process -FilePath "$script:baseUri/" })
$script:icon.Add_DoubleClick({ Start-Process -FilePath "$script:baseUri/" })
$script:restartItem.Add_Click({
    try {
        Start-Process -FilePath 'powershell.exe' -Verb RunAs -WindowStyle Hidden -ArgumentList @(
            '-NoProfile', '-Command', "Restart-Service -Name 'TorrServer-Flow'"
        ) | Out-Null
    } catch {
        [void][System.Windows.Forms.MessageBox]::Show($_.Exception.Message, 'TorrServer-Flow')
    }
})
$script:pauseItem.Add_Click({ Invoke-FlowAction 'pause' })
$script:resumeItem.Add_Click({ Invoke-FlowAction 'resume' })
$script:exitItem.Add_Click({
    $script:timer.Stop()
    $script:icon.Visible = $false
    $script:icon.Dispose()
    $script:menu.Dispose()
    $script:context.ExitThread()
})

try {
    Update-FlowTray
    $script:timer.Start()
    [System.Windows.Forms.Application]::Run($script:context)
} finally {
    $script:timer.Stop()
    $script:timer.Dispose()
    $script:icon.Visible = $false
    $script:icon.Dispose()
    $script:menu.Dispose()
    $script:context.Dispose()
}
