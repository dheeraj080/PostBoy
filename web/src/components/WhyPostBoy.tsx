import React from 'react';
import { HardDrive, Terminal, TerminalSquare } from 'lucide-react';
import { useInView } from '../hooks/useInView';

export function WhyPostBoy() {
  const [sectionRef, isInView] = useInView<HTMLElement>({ threshold: 0.1 });

  const benefits = [
    {
      title: 'LOCAL-FIRST',
      icon: HardDrive,
      codeBadge: '~/.config/postboy/',
      description:
        'Requests, collections, environments, history, and credentials stay on your machine.',
      detail: 'Pure local JSON and SQLite storage. Zero telemetry, zero cloud accounts, zero vendor lock-in.',
    },
    {
      title: 'TERMINAL-NATIVE',
      icon: Terminal,
      codeBadge: 'interactive TUI',
      description:
        'Build and inspect API requests without leaving your development environment.',
      detail: 'Streamlined navigation, tab cycling, external $EDITOR body editing, and instant response filtering.',
    },
    {
      title: 'SCRIPTABLE',
      icon: TerminalSquare,
      codeBadge: 'exit 0 / exit 1',
      description:
        'Turn the same collections into headless runs for CI and automation.',
      detail: 'Execute automated runs with status assertions and response data captures directly in shell scripts and CI.',
    },
  ];

  const delays = ['delay-0', 'delay-75', 'delay-150'];

  return (
    <section ref={sectionRef} className="py-14 border-b border-[#3E3E42] bg-[#1E1E1E]">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        {/* Differentiation Callout */}
        <div
          className={`mb-8 p-4 rounded-lg border border-[#3E3E42] bg-[#252526] text-xs font-mono text-[#CCCCCC] flex flex-col md:flex-row md:items-center justify-between gap-3 transition-all duration-300 ease-out ${
            isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
          }`}
        >
          <div className="flex items-center gap-2">
            <span className="text-[#FF6C37] font-bold">Purpose:</span>
            <span>curl is built for single commands. GUI clients rely on electron desktops.</span>
          </div>
          <div className="text-[#9CDCFE]">
            PostBoy gives you a fast, local API workflow that runs headlessly in CI.
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {benefits.map((b, idx) => {
            const Icon = b.icon;
            const delayClass = delays[idx] || 'delay-0';
            return (
              <div
                key={b.title}
                className={`p-5 rounded-lg border border-[#3E3E42] bg-[#252526]/50 hover:border-[#505054] flex flex-col justify-between transition-all duration-300 ease-out ${delayClass} ${
                  isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
                }`}
              >
                <div>
                  <div className="flex items-center justify-between mb-4">
                    <div className="w-8 h-8 rounded bg-[#1E1E1E] border border-[#3E3E42] flex items-center justify-center text-[#FF6C37]">
                      <Icon className="w-4 h-4" />
                    </div>
                    <span className="text-[11px] font-mono text-[#9CDCFE] bg-[#1E1E1E] px-2 py-0.5 rounded border border-[#3E3E42]">
                      {b.codeBadge}
                    </span>
                  </div>

                  <h2 className="text-sm font-mono font-bold tracking-wider text-white mb-2">
                    {b.title}
                  </h2>

                  <p className="text-sm font-sans text-[#CCCCCC] leading-relaxed mb-3">
                    {b.description}
                  </p>
                </div>

                <div className="pt-3 border-t border-[#3E3E42]/60 text-xs text-[#858585] font-sans">
                  {b.detail}
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
