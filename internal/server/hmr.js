(function () {
  if (
    window.__NOVA_HMR != null &&
    window.__NOVA_HMR instanceof EventSource &&
    window.__NOVA_HMR.readyState !== EventSource.CLOSED
  ) {
    return;
  }

  function getScript(file) {
    const path = file.startsWith('/') ? file : '/' + file;
    const scripts = Array.from(document.getElementsByTagName('script'));
    const script = scripts.find((s) => s.src.includes(location.origin + path));
    return script;
  }

  function updateScripts(files) {
    files.forEach((file) => {
      const script = getScript(file);
      if (!script) {
        return;
      }

      const next = document.createElement('script');
      next.type = 'module';
      next.src = script.src.split('?')[0] + '?' + Date.now();
      next.onload = () => {
        console.log(`[reloaded] ${file}`);
      };
      script.parentNode.replaceChild(next, script);
    });
  }

  function deleteScripts(files) {
    files.forEach((file) => {
      const script = getScript(file);
      if (script) {
        script.parentNode.removeChild(script);
      }
    });
  }

  function getLink(file) {
    const path = file.startsWith('/') ? file : '/' + file;
    const links = Array.from(document.getElementsByTagName('link'));
    const link = links.find((l) => l.href.includes(location.origin + path));
    return link;
  }

  function updateLinks(files) {
    files.forEach((file) => {
      const link = getLink(file);
      if (!link) {
        return;
      }

      const next = link.cloneNode();
      next.href = link.href.split('?')[0] + '?' + Date.now();
      next.onload = () => {
        console.log(`[reloaded] ${file}`);
      };
      link.parentNode.replaceChild(next, link.nextSibling);
    });
  }

  function deleteLinks(files) {
    files.forEach((file) => {
      const link = getLink(file);
      if (link) {
        link.parentElement.removeChild(link);
      }
    });
  }

  const sse = new EventSource('/@nova/hmr');
  sse.addEventListener('change', (event) => {
    const { created = [], deleted = [], updated = [] } = JSON.parse(event.data);

    console.log('[hmr]', { created }, { deleted }, { updated });
    if (
      updated.some((file) => file.endsWith('.go') || file.endsWith('.html'))
    ) {
      // Hot Reloading
      location.reload();
    } else {
      // Hot Module Replacement
      updateScripts(updated.filter((file) => file.endsWith('.js')));
      updateLinks(updated.filter((file) => file.endsWith('.css')));
    }

    deleteScripts(deleted.filter((file) => file.endsWith('.js')));
    deleteLinks(updated.filter((file) => file.endsWith('.css')));
  });
  window.__NOVA_HMR = sse;
})();
