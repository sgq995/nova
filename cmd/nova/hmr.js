function updateJS(files) {
  files.forEach((file) => {
    console.log(location.origin + file);
    const scripts = Array.from(document.getElementsByTagName('script'));
    const script = scripts.find((s) => s.src.includes(location.origin + file));
    console.log(script);
    if (script) {
      const next = document.createElement('script');
      next.src = script.src.split('?')[0] + '?' + Date.now();
      next.onload = () => {
        console.log(`[reloaded] ${file}`);
      };
      console.log(`[hmr] ${file}`);
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
      console.log(`[hmr] ${file}`);
      link.parentNode.replaceChild(next, link.nextSibling);
    }
  });
}

new EventSource('/esbuild').addEventListener('change', (event) => {
  const { added, removed, updated } = JSON.parse(event.data);

  updateJS(updated.filter((file) => file.endsWith('.js')));
  updateCSS(updated.filter((file) => file.endsWith('.css')));

  // TODO: verify conditions for location.reload
  // if (jsFiles.length > 0) {
  // } else {
  //   location.reload();
  // }
});
