import React from 'react';
import { Terminal, FileCode2, ArrowRightLeft } from 'lucide-react';
import { CopyButton } from './CopyButton';
import { useInView } from '../hooks/useInView';

export function ImportExport() {
  const [sectionRef, isInView] = useInView<HTMLElement>({ threshold: 0.1 });
  const importCli = 'postboy import collection.postman_collection.json';

  return (
    <section ref={sectionRef} className="py-20 border-b border-[#3E3E42] bg-[#1E1E1E]">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div
          className={`max-w-3xl mb-12 transition-all duration-300 ease-out ${
            isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
          }`}
        >
          <div className="text-xs font-mono uppercase tracking-wider text-[#FF6C37] font-semibold mb-2">
            Interoperability
          </div>
          <h2 className="text-2xl sm:text-3xl lg:text-4xl font-extrabold font-mono text-white tracking-tight">
            Bring your existing requests.
          </h2>
          <p className="mt-3 text-sm sm:text-base text-[#858585] font-sans">
            PostBoy works seamlessly with the tools your team already uses. Import and export curl commands and Postman v2.1 collections without data loss.
          </p>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
          {/* Left: Curl & Postman Cards */}
          <div
            className={`lg:col-span-7 space-y-6 transition-all duration-300 ease-out delay-75 ${
              isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
            }`}
          >
            {/* cURL Path */}
            <div className="p-5 rounded-lg border border-[#3E3E42] bg-[#252526] hover:border-[#505054] transition-colors">
              <div className="flex items-center gap-2 mb-3">
                <Terminal className="w-4 h-4 text-[#FF6C37]" />
                <h3 className="text-sm font-mono font-bold text-white">
                  curl Integration
                </h3>
              </div>

              <div className="space-y-4 text-xs font-sans">
                <div>
                  <div className="flex items-center justify-between mb-1.5 font-mono">
                    <span className="text-[#9CDCFE] font-bold">Import (Alt+I)</span>
                    <span className="text-[#858585] text-[11px]">POSIX bash, cmd.exe &amp; PowerShell</span>
                  </div>
                  <p className="text-[#CCCCCC] leading-relaxed mb-2">
                    Paste any curl command copied from browser DevTools or docs directly into PostBoy. Headers, URL, query params, auth, and bodies are parsed instantly.
                  </p>
                </div>

                <div className="pt-3 border-t border-[#3E3E42]">
                  <div className="flex items-center justify-between mb-1.5 font-mono">
                    <span className="text-[#4EC9B0] font-bold">Export (Alt+C)</span>
                    <span className="text-[#858585] text-[11px]">Clipboard &amp; OSC 52</span>
                  </div>
                  <p className="text-[#CCCCCC] leading-relaxed mb-2">
                    Copies your active request as a clean, multi-line POSIX curl command. Environment variables are expanded, while secrets remain referenced as <code className="text-[#9CDCFE]">{"{{ secret.NAME }}"}</code> so the snippet is safe to share.
                  </p>
                  <div className="p-2.5 rounded bg-[#1E1E1E] border border-[#3E3E42] font-mono text-[11px] text-[#9CDCFE] leading-relaxed overflow-x-auto select-text">
{`curl -X POST 'https://api.github.com/repos/owner/repo/issues' \\
  -H 'Authorization: Bearer {{ secret.GITHUB_PAT }}' \\
  -H 'Content-Type: application/json' \\
  --data-raw '{"title":"Bug report"}'`}
                  </div>
                </div>
              </div>
            </div>

            {/* Postman Path */}
            <div className="p-5 rounded-lg border border-[#3E3E42] bg-[#252526] hover:border-[#505054] transition-colors">
              <div className="flex items-center gap-2 mb-3">
                <FileCode2 className="w-4 h-4 text-[#4EC9B0]" />
                <h3 className="text-sm font-mono font-bold text-white">
                  Postman v2.1 Import &amp; Export
                </h3>
              </div>

              <div className="space-y-3 text-xs font-sans">
                <p className="text-[#CCCCCC] leading-relaxed">
                  Migrate existing team collections directly from the CLI or within the TUI. Postman variables are automatically imported as a new PostBoy environment.
                </p>

                <div className="p-3 rounded bg-[#1E1E1E] border border-[#3E3E42] flex items-center justify-between font-mono text-xs">
                  <span className="text-[#9CDCFE]">{importCli}</span>
                  <CopyButton text={importCli} />
                </div>

                <div className="pt-2 text-[11px] text-[#858585]">
                  <span className="font-semibold text-white">Export:</span> Press <kbd className="text-[#FF6C37] font-bold font-mono">e</kbd> inside the Collections modal (<kbd className="text-[#FF6C37] font-bold font-mono">Alt+O</kbd>) to generate a spec-compliant <code className="text-[#9CDCFE]">&lt;name&gt;.postman_collection.json</code> file.
                </div>
              </div>
            </div>
          </div>

          {/* Right: Visual Collection Tree */}
          <div
            className={`lg:col-span-5 transition-all duration-300 ease-out delay-150 ${
              isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-3'
            }`}
          >
            <div className="rounded-lg border border-[#3E3E42] bg-[#1E1E1E] shadow-xl overflow-hidden font-mono text-xs hover:border-[#505054] transition-colors">
              <div className="h-8 bg-[#252526] px-3.5 border-b border-[#3E3E42] flex items-center justify-between text-[#858585]">
                <div className="flex items-center gap-2">
                  <ArrowRightLeft className="w-3.5 h-3.5 text-[#FF6C37]" />
                  <span className="text-[11px] text-[#CCCCCC]">Collections Hierarchy</span>
                </div>
                <span className="text-[10px] text-[#4EC9B0]">collections.json</span>
              </div>

              <div className="p-4 bg-[#1E1E1E] space-y-4 text-[11px] leading-relaxed select-text">
                <div>
                  <div className="text-[#FF6C37] font-bold flex items-center gap-1.5 mb-1">
                    <span>▾</span>
                    <span>Authentication</span>
                    <span className="text-[#6E6E6E] font-normal">(3)</span>
                  </div>
                  <div className="pl-4 space-y-1 text-[#CCCCCC]">
                    <div className="flex items-center gap-2">
                      <span className="text-[#6E6E6E]">├──</span>
                      <span className="text-[#FF6C37] font-bold text-[10px] w-8">POST</span>
                      <span>Login</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-[#6E6E6E]">├──</span>
                      <span className="text-[#FF6C37] font-bold text-[10px] w-8">POST</span>
                      <span>Refresh token</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-[#6E6E6E]">└──</span>
                      <span className="text-[#4EC9B0] font-bold text-[10px] w-8">GET</span>
                      <span>Current user</span>
                    </div>
                  </div>
                </div>

                <div className="pt-2 border-t border-[#2D2D30]">
                  <div className="text-[#FF6C37] font-bold flex items-center gap-1.5 mb-1">
                    <span>▾</span>
                    <span>Users</span>
                    <span className="text-[#6E6E6E] font-normal">(3)</span>
                  </div>
                  <div className="pl-4 space-y-1 text-[#CCCCCC]">
                    <div className="flex items-center gap-2">
                      <span className="text-[#6E6E6E]">├──</span>
                      <span className="text-[#4EC9B0] font-bold text-[10px] w-8">GET</span>
                      <span>List users</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-[#6E6E6E]">├──</span>
                      <span className="text-[#FF6C37] font-bold text-[10px] w-8">POST</span>
                      <span>Create user</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-[#6E6E6E]">└──</span>
                      <span className="text-[#F48771] font-bold text-[10px] w-8">DELETE</span>
                      <span>Delete user</span>
                    </div>
                  </div>
                </div>

                <div className="pt-3 border-t border-[#2D2D30] text-[10px] text-[#858585] flex items-center justify-between">
                  <span>Enter: open • e: export • d: delete</span>
                  <span className="text-[#9CDCFE]">VCS-Friendly JSON</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
