const qs = s => document.querySelector(s);

async function init() {
    try {
        const response = await fetch('content.json');
        if (!response.ok) throw new Error('Network response was not ok');
        const data = await response.json();
        renderSite(data);
        setupInteractions();
    } catch (err) {
        console.error("Infrastructure Error:", err);
    }
}

function renderSite(data) {
    // 1. Metadata & Branding
    document.title = `${data.site.name} | ${data.site.tagline}`;
    const brandName = qs('#brand-name');
    if (brandName) brandName.textContent = data.site.name;
    document.querySelectorAll('.footer-brand').forEach(el => el.textContent = data.site.name);
    
    // 2. Navigation
    const navContainer = qs('#nav-links');
    if (navContainer) {
        navContainer.innerHTML = ''; 
        data.nav.forEach(item => {
            const a = document.createElement('a');
            a.href = item.href;
            a.textContent = item.label;
            a.className = 'hover:text-white transition-colors cursor-pointer';
            navContainer.appendChild(a);
        });
        
        // Navigation CTA for Email
        const navCta = document.createElement('button');
        navCta.className = 'bg-white text-black px-6 py-2.5 rounded-full hover:bg-blue-600 hover:text-white transition-all font-bold text-[10px] uppercase ml-4';
        navCta.textContent = 'Email Us';
        navCta.onclick = () => qs('#contact')?.scrollIntoView({ behavior: 'smooth' });
        navContainer.appendChild(navCta);
    }

    // 3. Hero Content
    const heroTagline = qs('#hero-tagline');
    const heroDesc = qs('#hero-desc');
    const ctaPrimary = qs('#cta-primary');
    const ctaSecondary = qs('#cta-secondary');

    if (heroTagline) heroTagline.textContent = data.site.tagline;
    if (heroDesc) heroDesc.textContent = data.site.description;
    
    if (ctaPrimary) {
        ctaPrimary.querySelector('span').textContent = `${data.hero.cta_primary}`;
        ctaPrimary.onclick = () => qs('#contact')?.scrollIntoView({ behavior: 'smooth' });
    }
    if (ctaSecondary) {
        ctaSecondary.textContent = `${data.hero.cta_secondary}`;
        ctaSecondary.onclick = () => {
            const trigger = qs('#chat-trigger');
            if (trigger) trigger.click();
        };
    }

    // 4. Values & Services Grids
    const valueGrid = qs('#value-grid');
    if (valueGrid && data.values) {
        valueGrid.innerHTML = '';
        data.values.forEach(v => {
            const div = document.createElement('div');
            div.className = 'glass-card rounded-[2.5rem] p-10 relative overflow-hidden group min-h-[400px] flex flex-col justify-between';
            div.innerHTML = `
                <div class="relative z-10">
                    <div class="flex-shrink-0 w-12 h-12 bg-blue-500/10 rounded-xl flex items-center justify-center text-blue-400 mb-8">
                        <i data-lucide="${v.icon}"></i>
                    </div>
                    <h4 class="text-3xl font-bold text-white mb-6 tracking-tight">${v.title}</h4>
                    <div class="space-y-6">
                        <div>
                            <span class="text-red-400/80 font-bold uppercase text-[10px] tracking-widest block mb-2">Problem</span>
                            <p class="text-gray-400 text-lg leading-relaxed">${v.problem}</p>
                        </div>
                        <div class="pt-4 border-t border-white/5">
                            <span class="text-emerald-400 font-bold uppercase text-[10px] tracking-widest block mb-2">Solution</span>
                            <p class="text-gray-200 text-lg leading-relaxed">${v.solution}</p>
                        </div>
                    </div>
                </div>
            `;
            valueGrid.appendChild(div);
        });
    }

    const servicesGrid = qs('#services-grid');
    if (servicesGrid) {
        servicesGrid.innerHTML = '';
        data.services.forEach(s => {
            const card = document.createElement('div');
            const accentClass = s.accent ? `text-${s.accent}-400` : 'text-blue-400';
            card.className = `glass-card rounded-[2.5rem] p-10 relative overflow-hidden group min-h-[400px] flex flex-col justify-between ${s.size || ''}`;
            card.innerHTML = `
                <div class="relative z-10">
                    <i data-lucide="${s.icon}" class="${accentClass} w-12 h-12 mb-8"></i>
                    <h4 class="text-3xl font-bold text-white mb-6 tracking-tight">${s.title}</h4>
                    <p class="text-gray-400 text-lg leading-relaxed max-w-md">${s.desc}</p>
                </div>
            `;
            servicesGrid.appendChild(card);
        });
    }

    // 5. Footer Rendering
    const footerBio = qs('#footer-bio');
    const footerCopy = qs('#footer-copy');
    const footerLinks = qs('#footer-links');

    if (footerBio) footerBio.textContent = data.site.description;
    if (footerCopy) footerCopy.textContent = data.footer.copyright;
    if (footerLinks) {
        footerLinks.innerHTML = '';
        data.footer.sections.forEach(section => {
            const div = document.createElement('div');
            const linksHtml = section.links.map(l => {
                const label = typeof l === 'object' ? l.label : l;
                const href = typeof l === 'object' ? l.href : '#';
                return `<a href="${href}" class="text-gray-500 hover:text-white transition-colors font-medium">${label}</a>`;
            }).join('');

            div.innerHTML = `<div class="text-white font-bold uppercase tracking-widest text-xs mb-6">${section.title}</div>
                             <div class="flex flex-col gap-4">${linksHtml}</div>`;
            footerLinks.appendChild(div);
        });
    }

    if (window.lucide) window.lucide.createIcons();
}

function setupInteractions() {
    // 1. Scroll Effects
    const nav = qs('#navbar');
    window.addEventListener('scroll', () => {
        if (!nav) return;
        const scrolled = window.scrollY > 50;
        nav.classList.toggle('nav-blur', scrolled);
        nav.classList.toggle('py-4', scrolled);
        nav.classList.toggle('py-8', !scrolled);
    });

    // 2. Email Form Logic
    const contactForm = qs('#contact-form');
    if (contactForm) {
        contactForm.addEventListener('submit', (e) => {
            e.preventDefault();
            const btn = contactForm.querySelector('button[type="submit"]');
            const originalText = btn.innerHTML;
            
            // Visual feedback for dispatch
            btn.innerHTML = '<i data-lucide="loader-2" class="animate-spin w-4 h-4 mr-2"></i> Sending Email...';
            btn.disabled = true;
            if (window.lucide) window.lucide.createIcons();

            // Simulate actual SMTP/API dispatch
            setTimeout(() => {
                btn.innerHTML = 'Email Sent Successfully';
                btn.classList.replace('bg-blue-600', 'bg-emerald-600');
                contactForm.reset();
                
                setTimeout(() => {
                    btn.innerHTML = originalText;
                    btn.disabled = false;
                    btn.classList.replace('bg-emerald-600', 'bg-blue-600');
                    if (window.lucide) window.lucide.createIcons();
                }, 4000);
            }, 2000);
        });
    }

    // 3. Chat Interaction Logic
    const chatTrigger = qs('#chat-trigger');
    const chatWindow = qs('#chat-window');
    const closeChat = qs('#close-chat');
    const sendChat = qs('#send-chat');
    const chatInput = qs('#chat-input');
    const chatMessages = qs('#chat-messages');

    if (chatTrigger && chatWindow) {
        const toggleChat = (forceOpen = false) => {
            const isHidden = chatWindow.classList.contains('hidden');
            if (isHidden || forceOpen) {
                chatWindow.classList.remove('hidden');
                requestAnimationFrame(() => {
                    chatWindow.classList.replace('translate-y-4', 'translate-y-0');
                    chatWindow.classList.replace('opacity-0', 'opacity-100');
                });
                
                // Focus input on open
                setTimeout(() => chatInput?.focus(), 400);
            } else {
                chatWindow.classList.replace('translate-y-0', 'translate-y-4');
                chatWindow.classList.replace('opacity-100', 'opacity-0');
                setTimeout(() => chatWindow.classList.add('hidden'), 300);
            }
        };

        chatTrigger.onclick = () => toggleChat();
        if (closeChat) closeChat.onclick = () => toggleChat();

        const appendMessage = (text, isUser = false) => {
            const msg = document.createElement('div');
            msg.className = isUser 
                ? 'bg-blue-600/20 p-3 rounded-2xl text-gray-200 max-w-[80%] self-end ml-auto' 
                : 'bg-white/5 p-3 rounded-2xl text-gray-300 max-w-[80%]';
            msg.textContent = text;
            chatMessages.appendChild(msg);
            chatMessages.scrollTop = chatMessages.scrollHeight;
        };

        const handleChatSend = () => {
            const val = chatInput.value.trim();
            if (!val) return;
            
            appendMessage(val, true);
            chatInput.value = '';

            // Simulate support engineer reply
            setTimeout(() => {
                appendMessage("System: Message delivered to on-call engineer. We've linked this session to your IP for follow-up.");
            }, 1000);
        };

        if (sendChat) sendChat.onclick = handleChatSend;
        if (chatInput) {
            chatInput.onkeypress = (e) => { if (e.key === 'Enter') handleChatSend(); };
        }
    }
}

// Initial execution
init();