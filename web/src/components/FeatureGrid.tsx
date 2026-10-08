import React from 'react';
import { 
  Send, 
  Workflow, 
  PlayCircle, 
  ArrowRightLeft 
} from 'lucide-react';
import { useInView } from '../hooks/useInView';

export function FeatureGrid() {
  const [sectionRef, isInView] = useInView<HTMLElement>({ threshold: 0.1 });

  const categories = [
    {
      id: 'requests',
      title: 'REQUESTS',
      icon: Send,
      summary: 'Craft, authenticate, and dispatch requests directly from the terminal.',
      items: [
        { label: 'HTTP Methods', detail: 'GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS (Alt+M to cycle).' },
        { label: 'Authentication', detail: 'Bearer Token, Basic Auth, and API Key (header or query param).' },
        { label: 'Headers & Params', detail: 'Key/value lists with instant toggle (t), edit, and delete.' },
        { label: 'Body Modes', detail: 'Raw with $EDITOR (Ctrl+O), Form URL-encoded, Multipart with files, and Binary File.' },
      ],
      code: `Alt+T  -> [Raw] Form URL-Encoded Multipart Binary
Ctrl+O -> edit raw body in $EDITOR (vi / notepad)`,
    },
    {
      id: 'workflows',
      title: 'WORKFLOWS',
      icon: Workflow,
      summary: 'Organize APIs locally with full environment templating and zero telemetry.',
      items: [
        { label: 'Collections', detail: 'Saved in human-readable collections.json with unsaved-change tracking.' },
        { label: 'Environments', detail: 'Named variable sets with dynamic active switching (Alt+E / Alt+V).' },
        { label: 'Variables', detail: 'Nested interpolation {{VAR}}, plus built-ins: {{$uuid}}, {{$timestamp}}, {{$randomInt}}.' },
        { label: 'Request History', detail: 'Local SQLite (postboy.db, up to 500 rows) with sensitive headers auto-redacted.' },
      ],
      code: `URL:  {{BASE_URL}}/orders/{{$uuid}}
Auth: Bearer {{ secret.API_TOKEN }} (OS Keychain)`,
    },
    {
      id: 'automation',
      title: 'AUTOMATION',
      icon: PlayCircle,
      summary: 'Turn interactive collections directly into automated test runs for scripts and CI.',
      items: [
        { label: 'Headless Runner', detail: 'postboy run "<collection>" --env <env> executes headlessly.' },
        { label: 'Status Assertions', detail: 'Declare expected_status to fail automatically on unexpected codes.' },
        { label: 'Response Captures', detail: 'Extract values with gjson paths and pass them to downstream requests.' },
        { label: 'Deterministic CI', detail: 'Exits 0 on all success, 1 on failure or 4xx/5xx. Flags: --continue, --verbose.' },
      ],
      code: `$ postboy run "Smoke Tests" --env Production
✓ #1 POST /login -> 200 (captured token)
✓ #2 GET /me     -> 200 (status 0: pass)`,
    },
    {
      id: 'interoperability',
      title: 'INTEROPERABILITY',
      icon: ArrowRightLeft,
      summary: 'Effortlessly bridge your team’s existing Postman collections and shell curl commands.',
      items: [
        { label: 'curl Import', detail: 'Alt+I parses curl commands from bash (\\), cmd.exe (^), or PowerShell (`).' },
        { label: 'curl Export', detail: 'Alt+C copies multi-line curl, leaving {{ secret.KEY }} unexpanded for safe sharing.' },
        { label: 'Postman Import', detail: 'postboy import <file> imports Postman v2.1 requests, folders, and environment vars.' },
        { label: 'Postman Export', detail: 'Export collections anytime with e in Collections (Alt+O).' },
      ],
      code: `$ postboy import stripe_api.postman_collection.json
✓ Imported 14 requests + created "stripe_api" env
Press 'e' in collections to export back to v2.1`,
    },
  ];

  // Stagger delays for the 4 architectural pillars
  const staggerDelays = ['delay-0', 'delay-75', 'delay-150', 'delay-200'];

  return (
    <section ref={sectionRef} id="features" className="py-20 border-b border-[#3E3E42] bg-[#1E1E1E]">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div
          className={`max-w-2xl mb-12 transition-all duration-300 ease-out ${
            isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
          }`}
        >
          <div className="text-xs font-mono uppercase tracking-wider text-[#FF6C37] font-semibold mb-2">
            Core Architecture
          </div>
          <h2 className="text-2xl sm:text-3xl font-extrabold font-mono text-white tracking-tight">
            Everything you need to work with APIs.
          </h2>
          <p className="mt-3 text-sm sm:text-base text-[#858585] font-sans">
            Organized across four foundational capabilities. Everything runs locally in your terminal.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {categories.map((cat, idx) => {
            const Icon = cat.icon;
            const delayClass = staggerDelays[idx] || 'delay-0';
            return (
              <div
                key={cat.id}
                className={`rounded-lg border border-[#3E3E42] bg-[#252526] p-6 flex flex-col justify-between hover:border-[#505054] transition-all duration-300 ease-out ${delayClass} ${
                  isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
                }`}
              >
                <div>
                  <div className="flex items-center gap-3 mb-3">
                    <div className="w-8 h-8 rounded bg-[#1E1E1E] border border-[#3E3E42] flex items-center justify-center text-[#FF6C37]">
                      <Icon className="w-4 h-4" />
                    </div>
                    <div>
                      <h3 className="text-sm font-mono font-bold tracking-wider text-white">
                        {cat.title}
                      </h3>
                      <p className="text-xs text-[#858585] font-sans">
                        {cat.summary}
                      </p>
                    </div>
                  </div>

                  <ul className="mt-4 space-y-2.5 text-xs font-sans divide-y divide-[#3E3E42]/40">
                    {cat.items.map((it) => (
                      <li key={it.label} className="pt-2.5 first:pt-0 flex flex-col sm:flex-row sm:items-baseline gap-1 sm:gap-2">
                        <span className="font-mono text-[#9CDCFE] font-semibold shrink-0">
                          {it.label}:
                        </span>
                        <span className="text-[#CCCCCC] leading-relaxed">
                          {it.detail}
                        </span>
                      </li>
                    ))}
                  </ul>
                </div>

                <div className="mt-5 pt-4 border-t border-[#3E3E42]">
                  <pre className="p-3 bg-[#1E1E1E] rounded border border-[#3E3E42]/80 font-mono text-[11px] text-[#4EC9B0] overflow-x-auto whitespace-pre leading-relaxed select-text">
                    {cat.code}
                  </pre>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
