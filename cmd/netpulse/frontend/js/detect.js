// ===== NetPulse Detection Actions =====

async function runPing(target, count) {
  const out = document.getElementById('detectOutput');
  out.innerHTML = '<span class="spinner"></span> 正在 Ping ' + target + ' ...';
  try {
    const r = await api(API.PING, { target, count });
    if (r.ok) {
      const d = r.data;
      out.innerHTML = '<span class="ok">✅ Ping ' + d.target + '</span>\n' +
        '发送: ' + d.sent + '  接收: ' + d.received + '  丢包: ' + d.packet_loss_pct.toFixed(1) + '%\n' +
        '平均延迟: <span class="' + (d.avg_rtt_ms < 30 ? 'ok' : d.avg_rtt_ms < 100 ? 'warn' : 'err') + '">' +
        d.avg_rtt_ms.toFixed(2) + ' ms</span>\n' +
        '耗时: ' + r.elapsed_ms + 'ms';
    } else {
      out.innerHTML = '<span class="err">❌ ' + r.error + '</span>';
    }
  } catch (e) {
    out.innerHTML = '<span class="err">❌ ' + e.message + '</span>';
  }
}

async function runDnsSpeed() {
  const out = document.getElementById('detectOutput');
  out.innerHTML = '<span class="spinner"></span> 正在测试 DNS 解析速度...';
  const domains = ['www.baidu.com', 'www.google.com', 'www.github.com', 'www.apple.com', 'www.bilibili.com'];
  try {
    const r = await api(API.DNS, { domains });
    if (r.ok) {
      const d = r.data;
      let html = '<span class="ok">🌐 DNS 解析测速</span>\n' + '─'.repeat(40) + '\n';
      for (const item of d.results) {
        const cls = item.error ? 'err' : item.latency_ms < 20 ? 'ok' : item.latency_ms < 80 ? 'warn' : 'err';
        html += item.domain.padEnd(25) + ' <span class="' + cls + '">' +
          (item.error || (item.latency_ms.toFixed(1) + ' ms')) + '</span>  ' + (item.ip || '') + '\n';
      }
      html += '─'.repeat(40) + '\n平均: <span class="info">' + d.avg_ms.toFixed(2) + ' ms</span>';
      out.innerHTML = html;
    } else {
      out.innerHTML = '<span class="err">❌ ' + r.error + '</span>';
    }
  } catch (e) {
    out.innerHTML = '<span class="err">❌ ' + e.message + '</span>';
  }
}

async function runTrace(target) {
  const out = document.getElementById('detectOutput');
  out.innerHTML = '<span class="spinner"></span> 正在追踪路由到 ' + target + ' ...（最多 15 跳，约 30 秒）';
  try {
    const r = await api(API.TRACE, { target });
    if (r.ok && r.data && r.data.hops && r.data.hops.length > 0) {
      const hops = r.data.hops;
      let html = '<span class="ok">🗺️ 路由追踪: ' + r.data.target + '</span>\n' + '─'.repeat(50) + '\n';
      for (const h of hops) {
        const ip = h.ip || '*';
        const rtt = h.rtt_ms ? h.rtt_ms.toFixed(1) + ' ms' : '*';
        html += String(h.hop).padStart(2) + '  ' + ip.padEnd(18) + '  ' + rtt + '\n';
      }
      html += '─'.repeat(50) + '\n共 ' + hops.length + ' 跳';
      out.innerHTML = html;
    } else if (r.error) {
      out.innerHTML = '<span class="err">❌ ' + r.error + '</span>\n\n<span class="warn">提示：部分网络环境（VPN/防火墙）会阻止路由追踪。</span>';
    } else {
      out.innerHTML = '<span class="warn">⚠️ 未收到路由数据\n\n提示：追踪超时或当前网络环境不支持（VPN/防火墙限制）。</span>';
    }
  } catch (e) {
    out.innerHTML = '<span class="err">❌ ' + e.message + '</span>';
  }
}
