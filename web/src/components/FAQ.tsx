import React, { useState } from 'react';
import { ChevronDown } from 'lucide-react';
import { useInView } from '../hooks/useInView';

export function FAQ() {
  const [sectionRef, isInView] = useInView<HTMLElement>({ threshold: 0.1 });
  const [openIdx, setOpenIdx] = useState<number | null>(0);

  const faqs = [
    {
      q: 'How are secrets stored?',
      a: 'Secret values are stored directly in your OS keychain (macOS Keychain, Linux Secret Service, Windows Credential Manager) via go-keyring. Only secret names are recorded in config.json. When running in headless environments without an OS keychain, secrets are held in memory only and discarded on process exit.',
    },
    {
      q: 'Can I import Postman collections?',
      a: 'Yes. Run "postboy import <collection.json>" or press Alt+I in the TUI to import Postman v2.1 collections. Requests, folders, headers, query params, auth, and bodies are preserved. Collection variables are converted into a PostBoy environment. You can also export back to Postman format with "e" in Collections.',
    },
    {
      q: 'Can I import curl commands?',
      a: 'Yes. Press Alt+I and paste any curl command (supports POSIX bash continuations, cmd.exe ^, and PowerShell `). To export, press Alt+C to copy a clean, multi-line curl command where secrets remain referenced as {{ secret.NAME }} so they are safe to share.',
    },
    {
      q: 'Can I run collections without the TUI?',
      a: 'Yes. Run "postboy run <Collection Name> --env <Environment>". It runs sequentially without launching the TUI, checks expected_status assertions, extracts response captures, and returns exit code 0 on success or 1 on failure.',
    },
    {
      q: 'Which platforms are supported?',
      a: 'PostBoy is compiled with CGO disabled (pure Go SQLite) and runs natively on Linux (amd64, arm64), macOS (Intel, Apple Silicon), and Windows (amd64, arm64).',
    },
    {
      q: 'Does PostBoy support WebSockets or gRPC?',
      a: 'No. PostBoy is designed strictly as an HTTP/1.1 and HTTP/2 client for REST and web APIs. It does not support WebSockets, gRPC, MQTT, or SSE streaming.',
    },
  ];

  return (
    <section ref={sectionRef} className="py-20 border-b border-[#3E3E42] bg-[#1E1E1E]">
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
        <div
          className={`max-w-2xl mb-10 transition-all duration-300 ease-out ${
            isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
          }`}
        >
          <div className="text-xs font-mono uppercase tracking-wider text-[#FF6C37] font-semibold mb-2">
            FAQ
          </div>
          <h2 className="text-2xl sm:text-3xl font-extrabold font-mono text-white tracking-tight">
            Frequently Asked Questions
          </h2>
          <p className="mt-2 text-sm text-[#858585] font-sans">
            Quick technical answers to common questions about PostBoy.
          </p>
        </div>

        <div
          className={`space-y-2.5 transition-all duration-300 ease-out delay-75 ${
            isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
          }`}
        >
          {faqs.map((faq, idx) => {
            const isOpen = openIdx === idx;
            return (
              <div
                key={faq.q}
                className="rounded-lg border border-[#3E3E42] bg-[#252526] hover:border-[#505054] overflow-hidden transition-colors"
              >
                <button
                  type="button"
                  id={`faq-btn-${idx}`}
                  onClick={() => setOpenIdx(isOpen ? null : idx)}
                  aria-expanded={isOpen}
                  aria-controls={`faq-panel-${idx}`}
                  className="w-full px-5 py-3.5 text-left flex items-center justify-between gap-4 focus:outline-none focus-visible:ring-1 focus-visible:ring-[#FF6C37] cursor-pointer select-none"
                >
                  <span className="font-mono text-sm font-semibold text-white">
                    {faq.q}
                  </span>
                  <ChevronDown
                    className={`w-4 h-4 text-[#858585] shrink-0 transition-transform duration-150 ease-out ${
                      isOpen ? 'rotate-180 text-[#FF6C37]' : ''
                    }`}
                  />
                </button>

                {/* Smooth Animated Height using CSS Grid */}
                <div
                  id={`faq-panel-${idx}`}
                  role="region"
                  aria-labelledby={`faq-btn-${idx}`}
                  className={`faq-grid ${isOpen ? 'is-open' : ''}`}
                >
                  <div className="faq-inner">
                    <div className="px-5 pb-4 pt-1.5 text-xs font-sans text-[#CCCCCC] leading-relaxed border-t border-[#3E3E42]/60 bg-[#1E1E1E]/40">
                      {faq.a}
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
