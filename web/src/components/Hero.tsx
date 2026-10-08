import React from 'react';
import { Terminal, ExternalLink, ArrowDown } from 'lucide-react';
import { TerminalSimulation } from './TerminalSimulation';

export function Hero() {
  const differentiators = [
    'Local & offline-first',
    'Terminal-native',
    'Collections run in CI',
    'Credentials in OS keychain',
  ];

  return (
    <section className="pt-10 pb-16 sm:pt-14 sm:pb-20 border-b border-[#3E3E42] bg-[#1E1E1E] overflow-hidden">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="max-w-3xl mb-8">  
          

          {/* 2. Primary headline (80ms) */}
          <h1 className="animate-hero-headline text-4xl sm:text-5xl lg:text-6xl font-extrabold tracking-tight text-white font-mono leading-tight mb-4 text-balance">
            API client,<br />
            <span className="text-[#FF6C37]">from the terminal.</span>
          </h1>

          {/* 3. Supporting description & differentiators (160ms) */}
          <div className="animate-hero-desc">
            <p className="text-base sm:text-lg text-[#CCCCCC] leading-relaxed mb-6 font-sans max-w-2xl">
              A local API client for developers who work in the terminal. Build requests, organize collections, manage environments and secrets, then run the same workflows in CI.
            </p>

            {/* 4 Core differentiators */}
            <div className="flex flex-wrap gap-2 sm:gap-3 mb-8">
              {differentiators.map((diff) => (
                <span
                  key={diff}
                  className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded bg-[#252526] border border-[#3E3E42] text-xs font-mono text-[#CCCCCC]"
                >
                  <span className="w-1.5 h-1.5 rounded-full bg-[#FF6C37]"></span>
                  <span>{diff}</span>
                </span>
              ))}
            </div>
          </div>

          {/* 4. CTAs (240ms) with polished hover and focus states */}
          <div className="animate-hero-cta flex flex-wrap items-center gap-3 sm:gap-4">
            <a
              href="#install"
              className="inline-flex items-center gap-2 px-5 py-2.5 rounded bg-[#FF6C37] hover:bg-[#FF7D4D] active:bg-[#e05b29] text-white font-mono font-semibold text-sm transition-all duration-150 ease-out hover:-translate-y-px active:translate-y-0 focus:outline-none focus-visible:ring-2 focus-visible:ring-[#FF6C37] focus-visible:ring-offset-2 focus-visible:ring-offset-[#1E1E1E] shadow-sm cursor-pointer select-none"
            >
              <Terminal className="w-4 h-4" />
              <span>Install with Go</span>
              <ArrowDown className="w-3.5 h-3.5 opacity-80" />
            </a>

            <a
              href="https://github.com/dheeraj080/PostBoy"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 px-5 py-2.5 rounded bg-[#252526] hover:bg-[#2e2e30] hover:text-white active:bg-[#252526] text-[#A0A0A0] border border-[#3E3E42] hover:border-[#505054] font-mono text-sm transition-all duration-150 ease-out hover:-translate-y-px active:translate-y-0 focus:outline-none focus-visible:ring-2 focus-visible:ring-[#9CDCFE] cursor-pointer select-none"
            >
              <span>View GitHub</span>
              <ExternalLink className="w-3.5 h-3.5 opacity-60" />
            </a>
          </div>
        </div>

        {/* 5. Primary Terminal Simulation Proof (320ms) */}
        <div className="animate-hero-terminal w-full">
          <TerminalSimulation />
        </div>
      </div>
    </section>
  );
}
