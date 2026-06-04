// NetPulse Global State

let dnsPresets = [];
let proxyEnabled = false;

async function loadPresets() {
  try {
    var r = await api(API.PRESETS);
    if (r.ok && Array.isArray(r.data)) {
      dnsPresets = r.data;
      renderDnsGrid();
    }
  } catch (e) {
    dnsPresets = [
      { name: '114DNS', servers: ['114.114.114.114', '114.114.115.115'] },
      { name: 'AliDNS', servers: ['223.5.5.5', '223.6.6.6'] },
      { name: 'DNSPod', servers: ['119.29.29.29', '119.28.28.28'] },
      { name: 'Cloudflare', servers: ['1.1.1.1', '1.0.0.1'] },
      { name: 'Google', servers: ['8.8.8.8', '8.8.4.4'] },
    ];
    renderDnsGrid();
  }
}

function rssiBars(rssi) {
  var bars = document.querySelectorAll('.signal-bars .bar');
  var count;
  if (rssi >= -50) count = 4;
  else if (rssi >= -65) count = 3;
  else if (rssi >= -75) count = 2;
  else if (rssi >= -85) count = 1;
  else count = 0;

  bars.forEach(function (b, i) {
    b.classList.remove('active', 'weak');
    if (i < count) {
      b.classList.add('active');
      if (count <= 2) b.classList.add('weak');
    }
  });
}

function metricColor(value, thresholds, unit) {
  // thresholds: [great, good] — below great=green, below good=amber, above=red
  if (value < 0) return '';
  if (value <= thresholds[0]) return 'green';
  if (value <= thresholds[1]) return 'amber';
  return 'red';
}

async function refreshAll() {
  try {
    var s = await api(API.STATUS);
    if (!s.ok) { setOffline(); return; }
    var d = s.data;

    // latency hero
    var latEl = document.getElementById('latVal');
    latEl.textContent = d.latency_ms > 0 ? d.latency_ms.toFixed(1) + ' ms' : '--';
    latEl.className = 'metric-value ' + metricColor(d.latency_ms, [30, 100]);
    document.getElementById('latSub').textContent =
      '丢包率 ' + (d.packet_loss_pct || 0).toFixed(1) + '%  ·  ' + (d.network_name || '');

    // DNS
    var dnsEl = document.getElementById('dnsVal');
    dnsEl.textContent = d.dns_speed_ms > 0 ? d.dns_speed_ms.toFixed(1) + ' ms' : '--';
    dnsEl.className = 'metric-value ' + metricColor(d.dns_speed_ms, [15, 60]);
    document.getElementById('dnsSub').textContent = d.dns_servers || '--';

    // WiFi
    var rssi = d.wifi_rssi || -100;
    var wEl = document.getElementById('wifiVal');
    wEl.textContent = d.wifi_ssid || '未连接';
    wEl.className = 'metric-value ' + (rssi > -65 ? 'green' : rssi > -78 ? 'amber' : 'red');
    document.getElementById('wifiSub').textContent =
      (d.wifi_rssi ? d.wifi_rssi + ' dBm' : '') + (d.wifi_noise ? '  /  ' + d.wifi_noise + ' dBm 噪声' : '');
    rssiBars(rssi);

    setOnline();
  } catch (e) {
    setOffline();
  }
}

function setOnline() {
  document.getElementById('globalDot').className = 'status-dot online';
  document.getElementById('globalStatus').textContent = '在线';
}

function setOffline() {
  document.getElementById('globalDot').className = 'status-dot offline';
  document.getElementById('globalStatus').textContent = '离线';
}
