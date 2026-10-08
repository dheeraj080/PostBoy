import React, { useState } from 'react';
import { Menu, X, Terminal } from 'lucide-react';

export function Header() {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const navLinks = [
    { label: 'Features', href: '#features' },
    { label: 'Quick Start', href: '#quickstart' },
    { label: 'CI', href: '#ci' },
    { label: 'Security', href: '#security' },
  ];

  return (
    <header className="sticky top-0 z-50 w-full border-b border-[#3E3E42] bg-[#1E1E1E]/95 backdrop-blur-sm">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-14 flex items-center justify-between">
        {/* Brand */}
        <a
          href="#"
          className="flex items-center gap-2.5 text-white font-mono font-bold tracking-tight text-lg group focus:outline-none focus-visible:ring-1 focus-visible:ring-[#FF6C37] rounded px-1.5 py-0.5 select-none"
        >
          <div className="w-6 h-6 rounded bg-[#252526] border border-[#3E3E42] flex items-center justify-center text-[#FF6C37] group-hover:border-[#FF6C37] transition-colors duration-150">
            <Terminal className="w-3.5 h-3.5" />
          </div>
          <span>
            Post<span className="text-[#FF6C37]">Boy</span>
          </span>
          <span className="text-[10px] uppercase tracking-wider font-semibold px-1.5 py-0.2 rounded bg-[#252526] text-[#9CDCFE] border border-[#3E3E42] hidden sm:inline-block">
            v1.0.0
          </span>
        </a>

        {/* Desktop Nav */}
        <nav className="hidden md:flex items-center gap-6" aria-label="Main Navigation">
          {navLinks.map((link) => (
            <a
              key={link.label}
              href={link.href}
              className="text-sm font-mono text-[#CCCCCC] hover:text-white transition-colors duration-150 focus:outline-none focus-visible:ring-1 focus-visible:ring-[#FF6C37] rounded px-1.5 py-0.5"
            >
              {link.label}
            </a>
          ))}
        </nav>

        {/* Mobile menu trigger */}
        <div className="flex md:hidden items-center gap-2">
          <a
            href="#install"
            className="text-xs font-mono font-semibold px-2.5 py-1 rounded bg-[#FF6C37] hover:bg-[#FF7D4D] active:bg-[#e05b29] text-white transition-all duration-150 ease-out hover:-translate-y-px active:translate-y-0 focus:outline-none focus-visible:ring-2 focus-visible:ring-[#FF6C37]"
          >
            Install
          </a>
          <button
            type="button"
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            className="p-1.5 text-[#CCCCCC] hover:text-white hover:bg-[#252526] rounded transition-colors duration-150 focus:outline-none focus-visible:ring-1 focus-visible:ring-[#FF6C37]"
            aria-expanded={mobileMenuOpen}
            aria-label="Toggle navigation menu"
          >
            {mobileMenuOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
          </button>
        </div>
      </div>

      {/* Mobile Nav dropdown */}
      {mobileMenuOpen && (
        <div className="md:hidden border-b border-[#3E3E42] bg-[#252526] px-4 pt-2 pb-4 space-y-2">
          {navLinks.map((link) => (
            <a
              key={link.label}
              href={link.href}
              onClick={() => setMobileMenuOpen(false)}
              className="block text-sm font-mono text-[#CCCCCC] hover:text-white hover:bg-[#1E1E1E] px-3 py-2 rounded transition-colors duration-150"
            >
              {link.label}
            </a>
          ))}
          <a
            href="https://github.com/dheeraj080/PostBoy"
            target="_blank"
            rel="noopener noreferrer"
            className="block text-sm font-mono text-[#9CDCFE] hover:text-white hover:bg-[#1E1E1E] px-3 py-2 rounded transition-colors duration-150"
          >
            GitHub Repository ↗
          </a>
        </div>
      )}
    </header>
  );
}
