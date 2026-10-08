import React from 'react';
import { CopyButton } from './CopyButton';
import { ArrowRight } from 'lucide-react';
import { useInView } from '../hooks/useInView';

export function HeadlessCI() {
  const [sectionRef, isInView] = useInView<HTMLElement>({ threshold: 0.15 });
  const runnerCmd = 'postboy run "Authentication Flow" --env Production';

  return (
    <section ref={sectionRef} id="ci" className="py-20 border-b border-[#3E3E42] bg-[#1E1E1E]">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div
          className={`max-w-3xl mb-8 transition-all duration-300 ease-out ${
            isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
          }`}
        >
          <div className="text-xs font-mono uppercase tracking-wider text-[#FF6C37] font-semibold mb-2">
            Automation &amp; CI
          </div>
          <h2 className="text-2xl sm:text-3xl lg:text-4xl font-extrabold font-mono text-white tracking-tight">
            Your interactive collections can become automation.
          </h2>
          <p className="mt-3 text-sm sm:text-base text-[#858585] font-sans">
            Run collections headlessly in scripts, pre-push git hooks, or CI pipelines without modifying a single request.
          </p>
        </div>

        {/* Transition statement */}
        <div
          className={`mb-8 p-3.5 rounded-lg border border-[#3E3E42] bg-[#252526] font-mono text-xs flex flex-wrap items-center gap-2 text-[#CCCCCC] transition-all duration-300 ease-out delay-75 ${
            isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
          }`}
        >
          <span className="text-[#FF6C37] font-semibold">Workflow transition:</span>
          <span>Build interactively in TUI</span>
          <ArrowRight className="w-3.5 h-3.5 text-[#858585]" />
          <span>Save to collections.json</span>
          <ArrowRight className="w-3.5 h-3.5 text-[#858585]" />
          <span className="text-[#4EC9B0] font-semibold">Run same collection headlessly</span>
        </div>

        {/* Large Code & Execution Visual */}
        <div
          className={`rounded-xl border border-[#3E3E42] bg-[#252526] p-4 sm:p-6 shadow-2xl transition-all duration-300 ease-out delay-100 ${
            isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
          }`}
        >
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-4 pb-4 border-b border-[#3E3E42]">
            <div className="flex items-center gap-2">
              <span className="w-2.5 h-2.5 rounded-full bg-[#4EC9B0]"></span>
              <span className="font-mono text-xs text-white font-bold">
                Headless Runner
              </span>
              <span className="text-[11px] font-mono text-[#858585]">
                (Zero TUI overhead)
              </span>
            </div>
            <CopyButton text={runnerCmd} label="Copy run command" />
          </div>

          {/* Terminal Command & Execution Output */}
          <div className="rounded-lg bg-[#1E1E1E] border border-[#3E3E42] p-4 sm:p-5 font-mono text-xs leading-relaxed select-text overflow-x-auto space-y-4">
            <div>
              <span className="text-[#6E6E6E] select-none">$ </span>
              <span className="text-[#9CDCFE] font-semibold">{runnerCmd}</span>
            </div>

            <div className="space-y-1.5 pt-1 text-[11px]">
              {/* Step 1: POST /login */}
              <div
                className={`transition-all duration-200 ease-out ${
                  isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-2'
                }`}
              >
                <div className="flex items-center gap-2 text-white">
                  <span className="text-[#4EC9B0] font-bold">✓</span>
                  <span className="text-[#858585]">#1</span>
                  <span className="text-[#FF6C37] font-bold">POST</span>
                  <span>/login</span>
                  <span className="text-[#6E6E6E]">-&gt;</span>
                  <span className="text-[#4EC9B0]">200 OK</span>
                  <span className="text-[#858585]">(118ms)</span>
                </div>
                <div className="pl-6 text-[10px] text-[#DCDCAA]">
                  ↳ captured token=&quot;eyJhbGciOiJIUzI1...&quot; from data.jwt
                </div>
              </div>

              {/* Step 2: GET /profile */}
              <div
                className={`transition-all duration-200 ease-out delay-150 ${
                  isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-2'
                }`}
              >
                <div className="flex items-center gap-2 text-white pt-1">
                  <span className="text-[#4EC9B0] font-bold">✓</span>
                  <span className="text-[#858585]">#2</span>
                  <span className="text-[#4EC9B0] font-bold">GET</span>
                  <span>/profile</span>
                  <span className="text-[#6E6E6E]">-&gt;</span>
                  <span className="text-[#4EC9B0]">200 OK</span>
                  <span className="text-[#858585]">(42ms)</span>
                  <span className="text-[10px] text-[#4EC9B0] bg-[#4EC9B0]/10 px-1 rounded">
                    expected_status 200 matched
                  </span>
                </div>
              </div>

              {/* Step 3: PUT /profile/preferences */}
              <div
                className={`transition-all duration-200 ease-out delay-300 ${
                  isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-2'
                }`}
              >
                <div className="flex items-center gap-2 text-white pt-1">
                  <span className="text-[#4EC9B0] font-bold">✓</span>
                  <span className="text-[#858585]">#3</span>
                  <span className="text-[#9CDCFE] font-bold">PUT</span>
                  <span>/profile/preferences</span>
                  <span className="text-[#6E6E6E]">-&gt;</span>
                  <span className="text-[#4EC9B0]">204 No Content</span>
                  <span className="text-[#858585]">(61ms)</span>
                </div>
              </div>

              {/* Summary & Exit Code */}
              <div
                className={`transition-all duration-200 ease-out delay-500 ${
                  isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-2'
                }`}
              >
                <div className="pt-3 border-t border-[#2D2D30] flex flex-wrap items-center justify-between gap-2 text-[#4EC9B0] font-bold">
                  <span>✓ 3 request(s) completed</span>
                  <span className="text-[#858585] font-normal text-[10px]">
                    Total time: 221ms
                  </span>
                </div>
                <div className="text-[#6E6E6E] text-[10px] mt-0.5">
                  exit 0 (ready for GitHub Actions / GitLab CI / shell scripts)
                </div>
              </div>
            </div>
          </div>

          {/* Highlights in 3 concise pillars */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 mt-4 pt-4 border-t border-[#3E3E42]/60 text-xs font-sans">
            <div className="p-2.5 rounded bg-[#1E1E1E] border border-[#3E3E42]/60 hover:border-[#505054] transition-colors">
              <span className="font-mono text-[#4EC9B0] font-bold block mb-1">
                expected_status
              </span>
              <span className="text-[#858585]">
                Assert exact HTTP status codes without writing test scripts.
              </span>
            </div>

            <div className="p-2.5 rounded bg-[#1E1E1E] border border-[#3E3E42]/60 hover:border-[#505054] transition-colors">
              <span className="font-mono text-[#9CDCFE] font-bold block mb-1">
                captures
              </span>
              <span className="text-[#858585]">
                Extract response JSON paths directly into runtime variables.
              </span>
            </div>

            <div className="p-2.5 rounded bg-[#1E1E1E] border border-[#3E3E42]/60 hover:border-[#505054] transition-colors">
              <span className="font-mono text-[#FF6C37] font-bold block mb-1">
                Exit Codes 0 / 1
              </span>
              <span className="text-[#858585]">
                Returns 0 on success, 1 on failures or 4xx/5xx responses.
              </span>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
