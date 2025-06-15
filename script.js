// Xelvra P2P Messenger Website JavaScript
// Modern, clean, and efficient

document.addEventListener('DOMContentLoaded', function() {
    // Initialize all components
    initNavigation();
    initScrollEffects();
    initTerminalAnimation();
    initCopyCode();
    initSmoothScrolling();
});

// Navigation functionality
function initNavigation() {
    const navToggle = document.getElementById('nav-toggle');
    const navMenu = document.getElementById('nav-menu');
    
    if (navToggle && navMenu) {
        navToggle.addEventListener('click', function() {
            navMenu.classList.toggle('active');
            navToggle.classList.toggle('active');
        });
        
        // Close menu when clicking on a link
        const navLinks = document.querySelectorAll('.nav-link');
        navLinks.forEach(link => {
            link.addEventListener('click', function() {
                navMenu.classList.remove('active');
                navToggle.classList.remove('active');
            });
        });
    }
    
    // Navbar scroll effect
    window.addEventListener('scroll', function() {
        const nav = document.querySelector('.nav');
        if (window.scrollY > 50) {
            nav.classList.add('scrolled');
        } else {
            nav.classList.remove('scrolled');
        }
    });
}

// Scroll effects for animations
function initScrollEffects() {
    const observerOptions = {
        threshold: 0.1,
        rootMargin: '0px 0px -50px 0px'
    };
    
    const observer = new IntersectionObserver(function(entries) {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.classList.add('animate-in');
            }
        });
    }, observerOptions);
    
    // Observe elements for animation
    const animateElements = document.querySelectorAll(
        '.feature-card, .tech-category, .epoch, .download-card'
    );

    animateElements.forEach(el => {
        observer.observe(el);
    });
}

// Terminal animation
function initTerminalAnimation() {
    const terminalLines = [
        { type: 'command', text: './peerchat-cli start', delay: 1000 },
        { type: 'output', text: '🚀 Starting Xelvra P2P Messenger CLI', delay: 2000 },
        { type: 'output', text: '✅ P2P node started successfully!', delay: 3000 },
        { type: 'output', text: '🆔 Your Peer ID: 12D3KooW...', delay: 4000 },
        { type: 'output', text: '📡 Listening on QUIC/TCP', delay: 5000 }
    ];
    
    const terminalBody = document.querySelector('.terminal-body');
    if (!terminalBody) return;
    
    // Clear existing content except the first line
    const firstLine = terminalBody.querySelector('.terminal-line');
    terminalBody.innerHTML = '';
    if (firstLine) {
        terminalBody.appendChild(firstLine);
    }
    
    let currentLine = 0;
    
    function addTerminalLine() {
        if (currentLine >= terminalLines.length) {
            // Add cursor line
            const cursorLine = document.createElement('div');
            cursorLine.className = 'terminal-line';
            cursorLine.innerHTML = '<span class="prompt">></span><span class="cursor">_</span>';
            terminalBody.appendChild(cursorLine);
            return;
        }
        
        const line = terminalLines[currentLine];
        const lineElement = document.createElement('div');
        lineElement.className = 'terminal-line';
        
        if (line.type === 'command') {
            lineElement.innerHTML = `<span class="prompt">$</span><span class="command">${line.text}</span>`;
        } else {
            lineElement.className += ' output';
            lineElement.innerHTML = `<span>${line.text}</span>`;
        }
        
        terminalBody.appendChild(lineElement);
        currentLine++;
        
        setTimeout(addTerminalLine, 1500);
    }
    
    // Start animation after a delay
    setTimeout(addTerminalLine, 2000);
}

// Copy code functionality
function initCopyCode() {
    window.copyCode = function() {
        const code = document.getElementById('quick-start-code');
        const copyBtn = document.querySelector('.copy-btn');
        
        if (code && copyBtn) {
            // Create a temporary textarea to copy the text
            const textarea = document.createElement('textarea');
            textarea.value = code.textContent;
            document.body.appendChild(textarea);
            textarea.select();
            
            try {
                document.execCommand('copy');
                copyBtn.textContent = 'Copied!';
                copyBtn.style.color = 'var(--color-accent)';
                
                setTimeout(() => {
                    copyBtn.textContent = 'Copy';
                    copyBtn.style.color = '';
                }, 2000);
            } catch (err) {
                console.error('Failed to copy code:', err);
            }
            
            document.body.removeChild(textarea);
        }
    };
}

// Smooth scrolling for navigation links
function initSmoothScrolling() {
    const navLinks = document.querySelectorAll('a[href^="#"]');
    
    navLinks.forEach(link => {
        link.addEventListener('click', function(e) {
            e.preventDefault();
            
            const targetId = this.getAttribute('href');
            const targetElement = document.querySelector(targetId);
            
            if (targetElement) {
                const offsetTop = targetElement.offsetTop - 80; // Account for fixed nav
                
                window.scrollTo({
                    top: offsetTop,
                    behavior: 'smooth'
                });
            }
        });
    });
}

// Add CSS classes for animations
const style = document.createElement('style');
style.textContent = `
    .nav.scrolled {
        background: rgba(10, 10, 10, 0.98);
        box-shadow: 0 2px 20px rgba(0, 0, 0, 0.3);
    }
    
    .feature-card,
    .tech-category,
    .epoch,
    .download-card {
        opacity: 0;
        transform: translateY(30px);
        transition: all 0.6s ease;
    }

    .feature-card.animate-in,
    .tech-category.animate-in,
    .epoch.animate-in,
    .download-card.animate-in {
        opacity: 1;
        transform: translateY(0);
    }
    
    @media (max-width: 768px) {
        .nav-menu {
            position: fixed;
            top: 70px;
            left: 0;
            right: 0;
            background: rgba(10, 10, 10, 0.98);
            backdrop-filter: blur(10px);
            border-bottom: 1px solid var(--color-border);
            flex-direction: column;
            padding: var(--spacing-md);
            transform: translateY(-100%);
            opacity: 0;
            transition: all 0.3s ease;
            pointer-events: none;
        }
        
        .nav-menu.active {
            display: flex;
            transform: translateY(0);
            opacity: 1;
            pointer-events: all;
        }
        
        .nav-toggle.active span:nth-child(1) {
            transform: rotate(45deg) translate(5px, 5px);
        }
        
        .nav-toggle.active span:nth-child(2) {
            opacity: 0;
        }
        
        .nav-toggle.active span:nth-child(3) {
            transform: rotate(-45deg) translate(7px, -6px);
        }
    }
`;
document.head.appendChild(style);

// Performance optimization: Throttle scroll events
function throttle(func, limit) {
    let inThrottle;
    return function() {
        const args = arguments;
        const context = this;
        if (!inThrottle) {
            func.apply(context, args);
            inThrottle = true;
            setTimeout(() => inThrottle = false, limit);
        }
    }
}

// Apply throttling to scroll events
window.addEventListener('scroll', throttle(function() {
    // Any scroll-based animations can be added here
}, 16)); // ~60fps

// Add loading animation
window.addEventListener('load', function() {
    document.body.classList.add('loaded');
});

// Add CSS for loading state
const loadingStyle = document.createElement('style');
loadingStyle.textContent = `
    body {
        opacity: 0;
        transition: opacity 0.5s ease;
    }
    
    body.loaded {
        opacity: 1;
    }
`;
document.head.appendChild(loadingStyle);
