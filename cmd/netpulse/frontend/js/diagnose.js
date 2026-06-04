// ===== NetPulse Diagnosis Engine =====

async function runDiagnose() {
  const out = document.getElementById('detectOutput');
  const diag = document.getElementById('diagnoseResults');
  out.innerHTML = '<span class="spinner"></span> 正在全面诊断网络...（约 5 秒）';
  diag.style.display = 'none';

  try {
    const r = await api(API.DIAGNOSE);
    if (!r.ok) { out.innerHTML = '<span class="err">❌ ' + r.error + '</span>'; return; }
    const d = r.data;

    out.innerHTML = '<span class="ok">✅ 诊断完成</span> 评分: ' + d.overall_score + '/100  ' +
      d.overall_verdict + '\n' + d.summary;

    const headerCls = d.overall_score >= 90 ? 'good' : d.overall_score >= 70 ? 'warn' : 'bad';
    let html = '<div class="diag-header ' + headerCls + '">' +
      '<div class="diag-score">' + d.overall_score + '</div>' +
      '<div>' +
        '<div class="diag-verdict">' + d.overall_verdict + '</div>' +
        '<div class="diag-summary">' + d.summary + '</div>' +
      '</div>' +
    '</div>';

    for (const f of d.findings) {
      const sevEmoji = f.severity === 'bad' ? '🔴' : f.severity === 'warn' ? '🟡' : f.severity === 'info' ? 'ℹ️' : '✅';
      html += '<div class="diag-finding">' +
        '<div class="sev">' + sevEmoji + '</div>' +
        '<div class="body">' +
          '<div class="title">' + f.title + '</div>' +
          '<div class="desc">' + f.description + '</div>' +
          (f.suggestion ? '<div class="sug">💡 ' + f.suggestion + '</div>' : '') +
          (f.action ? '<button class="fix-btn" onclick="applyFix(\'' + f.action + '\')">' + (f.actionLabel || '一键修复') + '</button>' : '') +
        '</div>' +
      '</div>';
    }

    html += '<div style="display:flex;justify-content:space-between;align-items:center;margin-top:' +
      'var(--space-2);font-size:var(--text-2xs);color:var(--color-text-secondary)">' +
      '<span>🕐 ' + d.checked_at + '</span>' +
      '<button class="btn btn-sm" onclick="runDiagnose()">🔄 重新诊断</button>' +
    '</div>';

    diag.innerHTML = html;
    diag.style.display = 'block';
  } catch (e) {
    out.innerHTML = '<span class="err">❌ ' + e.message + '</span>';
  }
}

async function applyFix(action) {
  if (action === 'flush-dns') {
    await flushDns();
    addLog('✅ 已执行 DNS 缓存刷新，3 秒后重新诊断...', 'ok');
    setTimeout(runDiagnose, 3000);
  } else if (action.startsWith('switch-dns:')) {
    const servers = action.replace('switch-dns:', '').split(',').map(function (s) { return s.trim(); });
    addLog('⚡ 正在切换 DNS...');
    try {
      const r = await api(API.DNS_SET, { servers: servers });
      if (r.ok) {
        addLog('✅ DNS 已切换: ' + servers.join(', '), 'ok');
        setTimeout(runDiagnose, 2000);
      } else {
        addLog('❌ ' + r.error, 'err');
      }
    } catch (e) {
      addLog('❌ ' + e.message, 'err');
    }
  }
}
