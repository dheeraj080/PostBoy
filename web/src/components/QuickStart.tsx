import React, { useState } from 'react';
import { ChevronDown, ArrowRight } from 'lucide-react';
import { useInView } from '../hooks/useInView';

export function QuickStart() {
  const [sectionRef, isInView] = useInView<HTMLElement>({ threshold: 0.1 });
  const [showAllShortcuts, setShowAllShortcuts] = useState(false);

  const workflowSteps = [
    { num: '1', label: 'Launch', desc: '$ postboy' },
    { num: '2', label: 'Configure', desc: 'Method, URL, body, auth' },
    { num: '3', label: 'Send', desc: 'Alt+R or Enter in URL' },
    { num: '4', label: 'Inspect', desc: '/ search · f filter · y copy' },
    { num: '5', label: 'Save', desc: 'Ctrl+S to collection' },
  ];

  const primaryShortcuts = [
    { key: 'Alt+R', desc: 'Send request' },
    { key: 'Ctrl+S', desc: 'Save request' },
    { key: 'Alt+O', desc: 'Open collections' },
    { key: 'Alt+V', desc: 'Environments' },
    { key: 'Alt+K', desc: 'Secrets manager' },
    { key: 'Alt+C', desc: 'Copy as curl' },
  ];

  const fullKeymap = [
    { key: 'Alt+X', desc: 'Cancel in-flight request' },
    { key: 'Alt+S', desc: 'Save request as copy' },
    { key: 'Alt+N', desc: 'New request (twice to discard)' },
    { key: 'Alt+I', desc: 'Import curl or Postman v2.1' },
    { key: 'Alt+M', desc: 'Cycle HTTP method' },
    { key: 'Alt+T', desc: 'Cycle request body mode' },
    { key: 'Alt+L', desc: 'Jump directly to Headers tab' },
    { key: 'Alt+H', desc: 'Open request history' },
    { key: 'Alt+E', desc: 'Cycle active environment' },
    { key: 'Tab / Shift+Tab', desc: 'Cycle panel focus' },
    { key: 'Ctrl+O', desc: 'Open raw body in $EDITOR' },
    { key: 'n / Enter / d', desc: 'Add / edit / delete in lists' },
    { key: 't / Space', desc: 'Toggle param / header on/off' },
    { key: 'v', desc: 'Toggle text vs file (multipart)' },
    { key: 'f', desc: 'Filter JSON response with gjson' },
    { key: 's', desc: 'Save response body to disk' },
  ];

  return (
    <section ref={sectionRef} id="quickstart" className="py-20 border-b border-[#3E3E42] bg-[#1E1E1E]">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div
          className={`max-w-2xl mb-12 transition-all duration-300 ease-out ${
            isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
          }`}
        >
          <div className="text-xs font-mono uppercase tracking-wider text-[#FF6C37] font-semibold mb-2">
            Workflow
          </div>
          <h2 className="text-2xl sm:text-3xl font-extrabold font-mono text-white tracking-tight">
            From terminal to response.
          </h2>
          <p className="mt-3 text-sm text-[#858585] font-sans">
            Start the TUI with a single command and drive your entire API lifecycle from the terminal.
          </p>
        </div>

        {/* Linear Workflow Bar */}
        <div
          className={`mb-12 rounded-lg border border-[#3E3E42] bg-[#252526] p-4 sm:p-6 transition-all duration-300 ease-out delay-75 ${
            isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
          }`}
        >
          <div className="grid grid-cols-1 sm:grid-cols-5 gap-3">
            {workflowSteps.map((step, idx) => (
              <div
                key={step.num}
                className="p-3 rounded bg-[#1E1E1E] border border-[#3E3E42]/80 hover:border-[#505054] transition-colors flex flex-col justify-between relative"
              >
                <div>
                  <div className="flex items-center justify-between mb-1.5">
                    <span className="text-[10px] font-mono text-[#FF6C37] font-bold">
                      STEP {step.num}
                    </span>
                    {idx < workflowSteps.length - 1 && (
                      <ArrowRight className="w-3 h-3 text-[#6E6E6E] hidden sm:block absolute -right-2 top-1/2 -translate-y-1/2 z-10" />
                    )}
                  </div>
                  <div className="text-xs font-mono font-bold text-white mb-0.5">
                    {step.label}
                  </div>
                  <div className="text-[11px] font-sans text-[#858585]">
                    {step.desc}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Core 6 Shortcuts */}
        <div
          className={`rounded-lg border border-[#3E3E42] bg-[#252526] p-6 max-w-4xl transition-all duration-300 ease-out delay-150 ${
            isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
          }`}
        >
          <div className="flex items-center justify-between mb-4 border-b border-[#3E3E42] pb-3">
            <div>
              <h3 className="text-sm font-mono font-bold text-white">
                Core shortcuts &amp; keymap
              </h3>
              <p className="text-xs text-[#858585] font-sans">
                Six primary chords drive 90% of requests.
              </p>
            </div>
            <button
              type="button"
              onClick={() => setShowAllShortcuts(!showAllShortcuts)}
              className="text-xs font-mono text-[#9CDCFE] hover:text-white flex items-center gap-1 transition-colors cursor-pointer focus:outline-none focus-visible:ring-1 focus-visible:ring-[#FF6C37] rounded px-1.5 py-0.5"
            >
              <span>{showAllShortcuts ? 'Collapse reference' : 'View all shortcuts'}</span>
              <ChevronDown className={`w-3.5 h-3.5 transition-transform duration-150 ${showAllShortcuts ? 'rotate-180' : ''}`} />
            </button>
          </div>

          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-6 gap-3">
            {primaryShortcuts.map((sc) => (
              <div
                key={sc.key}
                className="p-3 rounded bg-[#1E1E1E] border border-[#3E3E42] hover:border-[#505054] text-center flex flex-col items-center justify-center gap-1.5 transition-all duration-150 ease-out hover:-translate-y-px"
              >
                <kbd className="px-2 py-0.5 rounded bg-[#252526] border border-[#505054] text-[#FF6C37] font-mono font-bold text-xs tracking-tight shadow-sm">
                  {sc.key}
                </kbd>
                <span className="text-[11px] font-sans text-[#CCCCCC]">
                  {sc.desc}
                </span>
              </div>
            ))}
          </div>

          {/* Expandable Full Keymap */}
          {showAllShortcuts && (
            <div className="mt-6 pt-5 border-t border-[#3E3E42] transition-opacity duration-150">
              <div className="text-xs font-mono text-[#858585] mb-3">
                Complete TUI Keymap Reference
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-2.5">
                {fullKeymap.map((km) => (
                  <div key={km.key} className="p-2 rounded bg-[#1E1E1E] border border-[#3E3E42]/60 hover:border-[#505054] transition-colors text-xs flex items-baseline justify-between gap-2">
                    <kbd className="font-mono text-[#9CDCFE] font-semibold text-[11px] shrink-0">
                      {km.key}
                    </kbd>
                    <span className="text-[10px] text-[#858585] text-right truncate">
                      {km.desc}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
