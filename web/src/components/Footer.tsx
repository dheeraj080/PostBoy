import React from 'react';
import { Terminal, ExternalLink } from 'lucide-react';

export function Footer() {
  const footerLinks = [
    { label: 'GitHub', href: 'https://github.com/dheeraj080/PostBoy' },
    { label: 'README', href: 'https://github.com/dheeraj080/PostBoy#readme' },
    { label: 'Releases', href: 'https://github.com/dheeraj080/PostBoy/releases' },
    { label: 'Issues', href: 'https://github.com/dheeraj080/PostBoy/issues' },
    { label: 'License', href: 'https://github.com/dheeraj080/PostBoy/blob/main/LICENSE' },
  ];

  return (
    <footer className="bg-[#1A1A1A] text-[#858585] py-12 border-t border-[#3E3E42]">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-6">
          <div className="flex items-center gap-3">
            <div className="w-6 h-6 rounded bg-[#252526] border border-[#3E3E42] flex items-center justify-center text-[#FF6C37]">
              <Terminal className="w-3.5 h-3.5" />
            </div>
            <div>
              <span className="font-mono font-bold text-white text-sm">PostBoy</span>
              <span className="text-xs font-mono text-[#6E6E6E] ml-2">MIT License</span>
            </div>
          </div>

          <nav className="flex flex-wrap items-center gap-6" aria-label="Footer Navigation">
            {footerLinks.map((link) => (
              <a
                key={link.label}
                href={link.href}
                target="_blank"
                rel="noopener noreferrer"
                className="text-xs font-mono text-[#CCCCCC] hover:text-[#9CDCFE] transition-colors duration-150 inline-flex items-center gap-1 focus:outline-none focus-visible:ring-1 focus-visible:ring-[#FF6C37] rounded px-1"
              >
                <span>{link.label}</span>
                <ExternalLink className="w-3 h-3 text-[#6E6E6E]" />
              </a>
            ))}
          </nav>
        </div>

        <div className="mt-8 pt-6 border-t border-[#2D2D30] flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 text-xs font-mono text-[#6E6E6E]">
          <p>Terminal HTTP client and headless runner built in Go with Bubble Tea.</p>
          <p>No telemetry · No trackers · Open Source</p>
        </div>
      </div>
    </footer>
  );
}
