// ===== NetPulse Application Entry =====

document.addEventListener('DOMContentLoaded', function () {
  loadPresets().then(function () {
    refreshAll();
  });
});
