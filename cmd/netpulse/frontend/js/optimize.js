// ===== NetPulse Optimization Actions =====

function renderDnsGrid() {
  const grid = document.getElementById('dnsGrid');
  grid.innerHTML = dnsPresets.map(function (p) {
    var servers = Array.isArray(p.servers) ? p.servers.join(', ') : p.servers;
    return '<button class="dns-btn" onclick="setDns(\'' + servers + '\')">' +
      '<span class="name">' + p.name + '</span>' +
      '<span class="ip">' + servers + '</span>' +
    '</button>';
  }).join('');
}

async function setDns(serversStr) {
  const servers = serversStr.split(',').map(function (s) { return s.trim(); });
  addLog('📡 正在切换 DNS 到 ' + servers[0] + '...');
  try {
    const r = await api(API.DNS_SET, { servers: servers });
    if (r.ok) {
      addLog('✅ DNS 已切换', 'ok');
      refreshAll();
    } else {
      addLog('❌ ' + r.error, 'err');
    }
  } catch (e) {
    addLog('❌ ' + e.message, 'err');
  }
}

async function setCustomDns() {
  const val = document.getElementById('customDns').value.trim();
  if (!val) return;
  await setDns(val);
}

async function flushDns() {
  addLog('🚀 正在刷新 DNS 缓存...');
  try {
    const r = await api(API.FLUSH);
    if (r.ok) {
      addLog('✅ DNS 缓存已刷新', 'ok');
    } else {
      addLog('❌ ' + r.error, 'err');
    }
  } catch (e) {
    addLog('❌ ' + e.message, 'err');
  }
}

async function toggleProxyBtn() {
  proxyEnabled = !proxyEnabled;
  const btn = document.getElementById('proxyBtn');
  btn.textContent = proxyEnabled ? '📡 关闭代理' : '📡 开启代理';
  addLog((proxyEnabled ? '📡 正在开启' : '📡 正在关闭') + ' HTTP 代理...');
  try {
    const r = await api(API.PROXY, { enable: proxyEnabled, type: 'http' });
    if (r.ok) {
      addLog((proxyEnabled ? '✅ HTTP 代理已开启' : '✅ HTTP 代理已关闭'), 'ok');
    } else {
      addLog('❌ ' + r.error, 'err');
      proxyEnabled = !proxyEnabled;
    }
  } catch (e) {
    addLog('❌ ' + e.message, 'err');
    proxyEnabled = !proxyEnabled;
  }
}

function addLog(msg, cls) {
  const log = document.getElementById('logArea');
  const now = new Date();
  const time = now.getHours().toString().padStart(2, '0') + ':' +
               now.getMinutes().toString().padStart(2, '0') + ':' +
               now.getSeconds().toString().padStart(2, '0');
  const clsName = cls === 'ok' ? 'log-ok' : cls === 'err' ? 'log-err' : '';
  log.innerHTML += '<div class="log-entry"><span class="log-time">' + time + '</span><span class="' + clsName + '">' + msg + '</span></div>';
  log.scrollTop = log.scrollHeight;
}
