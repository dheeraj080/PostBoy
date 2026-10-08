import React, { useState } from 'react';
import { Terminal, ChevronRight, ExternalLink } from 'lucide-react';
import { CopyButton } from './CopyButton';
import { useInView } from '../hooks/useInView';

export function Installation() {
  const [sectionRef, isInView] = useInView<HTMLElement>({ threshold: 0.1 });
  const [showSource, setShowSource] = useState(false);

  const goCmd = 'go install github.com/dheeraj080/PostBoy/cmd/postboy@latest';
  const verifyCmd = 'postboy --version';
  const sourceCmds = `git clone https://github.com/dheeraj080/PostBoy.git
cd PostBoy
go build -o postboy ./cmd/postboy`;

  return (
    <section ref={sectionRef} id="install" className="py-20 border-b border-[#3E3E42] bg-[#1E1E1E]">
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
        <div
          className={`text-center max-w-2xl mx-auto mb-10 transition-all duration-300 ease-out ${
            isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
          }`}
        >
          <div className="text-xs font-mono uppercase tracking-wider text-[#FF6C37] font-semibold mb-2">
            Installation
          </div>
          <h2 className="text-3xl sm:text-4xl font-extrabold font-mono text-white tracking-tight">
            Install in seconds.
          </h2>
          <p className="mt-3 text-sm text-[#858585] font-sans">
            One command places the standalone binary directly on your system PATH.
          </p>
        </div>

        {/* Primary Installation Block */}
        <div
          className={`rounded-xl border border-[#3E3E42] bg-[#252526] p-6 sm:p-8 shadow-2xl space-y-6 transition-all duration-300 ease-out delay-75 ${
            isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
          }`}
        >
          {/* Main Install Command */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-mono text-[#CCCCCC] font-semibold flex items-center gap-1.5">
                <Terminal className="w-3.5 h-3.5 text-[#FF6C37]" />
                <span>Primary installation</span>
              </span>
              <CopyButton text={goCmd} label="Copy command" />
            </div>

            <div className="p-3.5 sm:p-4 bg-[#1E1E1E] rounded-lg border border-[#3E3E42] font-mono text-xs sm:text-sm text-white overflow-x-auto flex items-center justify-between gap-3 select-text">
              <div className="flex items-center gap-2">
                <span className="text-[#6E6E6E] select-none">$</span>
                <span className="text-[#9CDCFE]">{goCmd}</span>
              </div>
            </div>

            <div className="mt-2.5 flex items-center gap-2 text-xs font-mono text-[#858585]">
              <span className="w-1.5 h-1.5 rounded-full bg-[#4EC9B0]"></span>
              <span>Requires Go 1.22+</span>
            </div>
          </div>

          {/* Verification Command */}
          <div className="pt-5 border-t border-[#3E3E42]/80">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-mono text-[#858585]">
                Verify installation
              </span>
              <CopyButton text={verifyCmd} />
            </div>

            <div className="p-3 bg-[#1E1E1E] rounded-lg border border-[#3E3E42] font-mono text-xs text-[#9CDCFE] overflow-x-auto select-text">
              <span className="text-[#6E6E6E] select-none">$ </span>{verifyCmd}
              <div className="text-[#858585] text-[11px] mt-1 select-none">
                postboy v1.0.0 (commit 35b1b49, built 2026-10-08)
              </div>
            </div>
          </div>

          {/* Secondary Options */}
          <div className="pt-4 border-t border-[#3E3E42]/60 flex flex-col sm:flex-row sm:items-center justify-between gap-4 text-xs font-mono">
            {/* Quieter Binary Link */}
            <div className="flex items-center gap-1.5 text-[#858585]">
              <span>Prefer binaries?</span>
              <a
                href="https://github.com/dheeraj080/PostBoy/releases"
                target="_blank"
                rel="noopener noreferrer"
                className="text-[#9CDCFE] hover:text-white underline underline-offset-4 flex items-center gap-1 transition-colors"
              >
                <span>View GitHub Releases</span>
                <ExternalLink className="w-3 h-3" />
              </a>
              <span className="text-[#6E6E6E] text-[11px] hidden sm:inline">(Linux, macOS, Windows)</span>
            </div>

            {/* Collapsible Source Option */}
            <button
              type="button"
              onClick={() => setShowSource(!showSource)}
              className="text-[#858585] hover:text-[#CCCCCC] flex items-center gap-1 text-left transition-colors self-start sm:self-auto cursor-pointer focus:outline-none focus-visible:ring-1 focus-visible:ring-[#FF6C37] rounded px-1 py-0.5"
            >
              <span>Build from source</span>
              <ChevronRight className={`w-3.5 h-3.5 transition-transform duration-150 ${showSource ? 'rotate-90 text-[#FF6C37]' : ''}`} />
            </button>
          </div>

          {/* Collapsible Source Code Area */}
          {showSource && (
            <div className="pt-4 border-t border-[#3E3E42]/60 transition-all duration-200">
              <div className="flex items-center justify-between mb-2">
                <span className="text-xs font-mono text-[#858585]">Compile with Go toolchain</span>
                <CopyButton text={sourceCmds} />
              </div>
              <pre className="p-3 bg-[#1E1E1E] rounded border border-[#3E3E42] font-mono text-xs text-[#9CDCFE] overflow-x-auto select-text leading-relaxed">
{sourceCmds}
              </pre>
            </div>
          )}
        </div>

        {/* Verified No-Homebrew reassurance note */}
        <div className="mt-4 text-center text-xs text-[#6E6E6E] font-sans">
          No Homebrew tap is published. Install directly using Go or grab the official standalone binaries from GitHub Releases.
        </div>
      </div>
    </section>
  );
}
