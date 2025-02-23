(function () {
  if (
    window.__NOVA_HMR != null &&
    window.__NOVA_HMR instanceof EventSource &&
    window.__NOVA_HMR.readyState !== EventSource.CLOSED
  ) {
    return;
  }

  function updateJS(files) {
    files.forEach((file) => {
      const scripts = Array.from(document.getElementsByTagName('script'));
      const script = scripts.find((s) =>
        s.src.includes(location.origin + file)
      );
      if (script) {
        const next = document.createElement('script');
        next.src = script.src.split('?')[0] + '?' + Date.now();
        next.onload = () => {
          console.log(`[reloaded] ${file}`);
        };
        script.parentNode.replaceChild(next, script);
      }
    });
  }

  function updateCSS(files) {
    files.forEach((file) => {
      const links = Array.from(document.getElementsByTagName('link'));
      const link = links.find((l) => l.href.includes(location.origin + file));
      if (link) {
        const next = link.cloneNode();
        next.href = link.href.split('?')[0] + '?' + Date.now();
        next.onload = () => {
          console.log(`[reloaded] ${file}`);
        };
        link.parentNode.replaceChild(next, link.nextSibling);
      }
    });
  }

  const sse = new EventSource('/esbuild');
  sse.readyState;
  sse.addEventListener('change', (event) => {
    const { added, removed, updated } = JSON.parse(event.data);

    console.log('[hmr]', { added }, { removed }, { updated });
    updateJS(updated.filter((file) => file.endsWith('.js')));
    updateCSS(updated.filter((file) => file.endsWith('.css')));

    // TODO: verify conditions for location.reload
    // if (jsFiles.length > 0) {
    // } else {
    //   location.reload();
    // }
  });
  window.__NOVA_HMR = sse;
})();
