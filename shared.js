// shared.js — generated from content/site.toml by read_site node
// Injects nav, footer, and scroll behaviour into every page

(function() {
  const loadFragment = async (selector, url, callback) => {
    const placeholder = document.querySelector(selector);
    if (!placeholder) return;

    try {
      const response = await fetch(url);
      if (!response.ok) return;
      placeholder.innerHTML = await response.text();
      if (typeof callback === 'function') callback();
    } catch (error) {
      console.warn(`Unable to load fragment: ${url}`, error);
    }
  };

  const setupNav = () => {
    const nav = document.getElementById('nav');
    if (nav) {
      window.addEventListener('scroll', () => {
        nav.classList.toggle('scrolled', window.scrollY > 20);
      }, { passive: true });
    }

    const path = window.location.pathname;
    document.querySelectorAll('.nav-links a').forEach(a => {
      const li = a.parentElement;
      if (a.getAttribute('href') === path) li.classList.add('active');
    });
  };

  const setupReveal = () => {
    const reveals = document.querySelectorAll('.reveal');
    if (!reveals.length) return;

    const observer = new IntersectionObserver((entries) => {
      entries.forEach((entry, i) => {
        if (entry.isIntersecting) {
          setTimeout(() => entry.target.classList.add('visible'), i * 80);
          observer.unobserve(entry.target);
        }
      });
    }, { threshold: 0.08 });

    reveals.forEach(el => observer.observe(el));
  };

  document.addEventListener('DOMContentLoaded', () => {
    loadFragment('#nav-placeholder', '/partials/nav.html', setupNav);
    loadFragment('#footer-placeholder', '/partials/footer.html');
    setupReveal();
  });
})();