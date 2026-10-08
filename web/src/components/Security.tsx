import React from 'react';
import { ShieldCheck, Lock, EyeOff } from 'lucide-react';
import { useInView } from '../hooks/useInView';

export function Security() {
  const [sectionRef, isInView] = useInView<HTMLElement>({ threshold: 0.15 });

  return (
    <section ref={sectionRef} id="security" className="py-20 border-b border-[#3E3E42] bg-[#1E1E1E]">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div
          className={`max-w-3xl mb-10 transition-all duration-300 ease-out ${
            isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
          }`}
        >
          <div className="text-xs font-mono uppercase tracking-wider text-[#FF6C37] font-semibold mb-2">
            Storage &amp; Credentials
          </div>
          <h2 className="text-2xl sm:text-3xl lg:text-4xl font-extrabold font-mono text-white tracking-tight">
            Your credentials stay yours.
          </h2>
          <p className="mt-3 text-sm sm:text-base text-[#858585] font-sans">
            Secret values are never stored in PostBoy&apos;s configuration files or git history.
          </p>
        </div>

        {/* Visual Architecture Diagram */}
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
          {/* Left: Terminal Architecture Diagram with Staged Reveal */}
          <div className="lg:col-span-7">
            <div
              className={`rounded-lg border border-[#3E3E42] bg-[#1E1E1E] shadow-xl overflow-hidden font-mono text-xs select-text transition-all duration-300 ease-out delay-75 ${
                isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
              }`}
            >
              <div className="h-8 bg-[#252526] px-3.5 border-b border-[#3E3E42] flex items-center justify-between text-[#858585]">
                <div className="flex items-center gap-2">
                  <Lock className="w-3.5 h-3.5 text-[#FF6C37]" />
                  <span className="text-[11px] text-[#CCCCCC]">Local Storage &amp; Secret Flow</span>
                </div>
                <span className="text-[10px] text-[#4EC9B0]">0600 file mode</span>
              </div>

              <div className="p-4 sm:p-5 bg-[#1E1E1E] leading-relaxed text-[#CCCCCC] overflow-x-auto font-mono">
                {/* Stage 1: Sources */}
                <div
                  className={`text-[11px] text-[#9CDCFE] transition-all duration-200 ease-out ${
                    isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-2'
                  }`}
                >
                  <pre className="leading-relaxed">
{`config.json       ──▶ Secret NAMES only (never plain values)
collections.json  ──▶ Requests + variables
postboy.db        ──▶ Local SQLite history ([REDACTED] headers)
OS Keychain       ──▶ Actual secret values (macOS / Linux / Windows)`}
                  </pre>
                </div>

                {/* Stage 2: PostBoy engine node */}
                <div
                  className={`text-[11px] text-[#9CDCFE] transition-all duration-200 ease-out delay-150 ${
                    isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-2'
                  }`}
                >
                  <pre className="leading-relaxed">
{`       │
       ▼
┌───────────────┐
│    PostBoy    │  ──▶ Resolves secrets only in memory at send time
└───────┬───────┘`}
                  </pre>
                </div>

                {/* Stage 3: Outbound HTTP Request */}
                <div
                  className={`text-[11px] text-[#9CDCFE] transition-all duration-200 ease-out delay-300 ${
                    isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-2'
                  }`}
                >
                  <pre className="leading-relaxed">
{`        │
        ▼
   HTTP Request    ──▶ Dispatched over TLS (insecure_skip_verify is opt-in)`}
                  </pre>
                </div>

                <div className="mt-3 pt-3 border-t border-[#2D2D30] text-[11px] text-[#DCDCAA] space-y-1">
                  <div>
                    <span className="text-[#FF6C37] font-bold">Safe reference: </span>
                    <code>&#123;&#123; secret.GITHUB_TOKEN &#125;&#125;</code>
                  </div>
                  <div className="text-[#858585] text-[10px]">
                    ↳ Never written to disk. If no keychain is available (e.g. headless CI), secrets stay in memory only with a clear startup warning.
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Right: Technical Verification Points */}
          <div className="lg:col-span-5 space-y-4">
            <div
              className={`p-4 rounded-lg border border-[#3E3E42] bg-[#252526] hover:border-[#505054] transition-all duration-300 ease-out delay-75 ${
                isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
              }`}
            >
              <div className="flex items-center gap-2 mb-2 text-white font-mono font-bold text-xs">
                <ShieldCheck className="w-4 h-4 text-[#4EC9B0]" />
                <span>OS Keychain Storage</span>
              </div>
              <p className="text-xs text-[#CCCCCC] font-sans leading-relaxed">
                Tokens and passwords live in macOS Keychain, Windows Credential Manager, or Linux Secret Service (service: <code className="text-[#9CDCFE]">postboy</code>).
              </p>
            </div>

            <div
              className={`p-4 rounded-lg border border-[#3E3E42] bg-[#252526] hover:border-[#505054] transition-all duration-300 ease-out delay-150 ${
                isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
              }`}
            >
              <div className="flex items-center gap-2 mb-2 text-white font-mono font-bold text-xs">
                <EyeOff className="w-4 h-4 text-[#FF6C37]" />
                <span>Automatic Header Redaction</span>
              </div>
              <p className="text-xs text-[#CCCCCC] font-sans leading-relaxed">
                Sensitive headers (<code className="text-[#9CDCFE]">Authorization</code>, <code className="text-[#9CDCFE]">Cookie</code>, <code className="text-[#9CDCFE]">X-API-Key</code>) are automatically masked as <code className="text-[#DCDCAA]">[REDACTED]</code> before persisting to SQLite history.
              </p>
            </div>

            <div
              className={`p-4 rounded-lg border border-[#3E3E42] bg-[#252526] hover:border-[#505054] transition-all duration-300 ease-out delay-200 ${
                isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
              }`}
            >
              <div className="flex items-center gap-2 mb-2 text-white font-mono font-bold text-xs">
                <Lock className="w-4 h-4 text-[#9CDCFE]" />
                <span>Zero Telemetry &amp; Local Files</span>
              </div>
              <p className="text-xs text-[#CCCCCC] font-sans leading-relaxed">
                Zero analytics, zero tracking, and zero external servers. Files are written with restricted 0600 permissions, accessible only by your user account.
              </p>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
