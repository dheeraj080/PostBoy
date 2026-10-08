import React, { useState, useCallback } from 'react';
import { useInView } from '../hooks/useInView';

type ReqTab = 'body' | 'headers' | 'params' | 'auth';
type ResTab = 'body' | 'headers';

export function TerminalSimulation() {
  const [termRef, isInView] = useInView<HTMLDivElement>({ threshold: 0.15 });
  const [reqTab, setReqTab] = useState<ReqTab>('body');
  const [resTab, setResTab] = useState<ResTab>('body');
  const [isLoading, setIsLoading] = useState(false);
  const [activeFilter, setActiveFilter] = useState('');
  const [isFilterActive, setIsFilterActive] = useState(false);
  const [isRawMode, setIsRawMode] = useState(false);
  const [statusMsg, setStatusMsg] = useState('201 Created • 142ms');
  const [statusIsErr, setStatusIsErr] = useState(false);

  const handleSend = useCallback(() => {
    setIsLoading(true);
    setStatusMsg('Sending...');
    const timer = setTimeout(() => {
      setIsLoading(false);
      setStatusMsg('201 Created • 138ms');
      setStatusIsErr(false);
    }, 450);
    return () => clearTimeout(timer);
  }, []);

  const handleFilterToggle = useCallback((filterVal: string) => {
    if (activeFilter === filterVal) {
      setActiveFilter('');
      setIsFilterActive(false);
      setStatusMsg('Filter cleared');
    } else {
      setActiveFilter(filterVal);
      setIsFilterActive(true);
      setStatusMsg(`Filter applied: ${filterVal} • Esc to clear`);
    }
  }, [activeFilter]);

  return (
    <div
      ref={termRef}
      className="w-full rounded-lg border border-[#3E3E42] bg-[#1E1E1E] shadow-2xl overflow-hidden font-mono text-xs select-text"
      role="region"
      aria-label="Interactive Terminal Simulation"
    >
      {/* Terminal Title Bar / Window Header */}
      <div className="h-8 bg-[#252526] px-3.5 border-b border-[#3E3E42] flex items-center justify-between text-[#858585] select-none">
        <div className="flex items-center gap-2">
          <div className="w-2.5 h-2.5 rounded-full bg-[#3E3E42]"></div>
          <div className="w-2.5 h-2.5 rounded-full bg-[#3E3E42]"></div>
          <div className="w-2.5 h-2.5 rounded-full bg-[#3E3E42]"></div>
          <span className="ml-2 text-[11px] text-[#CCCCCC]">postboy — terminal HTTP client</span>
        </div>
        <div className="flex items-center gap-2 text-[11px] text-[#6E6E6E]">
          <span>80×24</span>
        </div>
      </div>

      {/* PostBoy Top Bar / Request Bar (Stage 1) */}
      <div
        className={`p-2 sm:p-2.5 bg-[#1E1E1E] border-b border-[#252526] flex items-center gap-2 transition-all duration-300 ease-out ${
          isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-2'
        }`}
      >
        <span className="text-[#6E6E6E] font-bold hidden sm:inline select-none">PostBoy</span>
        
        <span className="text-[#FF6C37] font-bold bg-[#252526] px-2 py-0.5 rounded border border-[#3E3E42] text-[11px] shrink-0 select-none">
          [POST]
        </span>

        {/* URL Input */}
        <div className="flex-1 min-w-0 bg-[#252526] text-[#CCCCCC] px-2.5 py-1 rounded border border-[#FF6C37]/60 flex items-center gap-1 overflow-hidden transition-colors focus-within:border-[#FF6C37]">
          <span className="text-[#FF6C37] select-none text-[11px] shrink-0">https://</span>
          <span className="text-white text-[11px] truncate">api.github.com/repos/owner/api/issues</span>
          <span className="inline-block w-1.5 h-3 bg-[#FF6C37] animate-pulse ml-0.5 shrink-0" aria-hidden="true"></span>
        </div>

        {/* Send Button */}
        <button
          onClick={handleSend}
          disabled={isLoading}
          type="button"
          className="bg-[#FF6C37] hover:bg-[#FF7D4D] active:bg-[#e05b29] text-white px-2.5 sm:px-3 py-1 rounded font-bold transition-all duration-150 ease-out hover:-translate-y-px active:translate-y-0 cursor-pointer disabled:opacity-50 text-[11px] shrink-0 focus:outline-none focus-visible:ring-2 focus-visible:ring-[#FF6C37] focus-visible:ring-offset-1 focus-visible:ring-offset-[#1E1E1E]"
          title="Click to trigger mock request (Alt+R)"
        >
          {isLoading ? 'Loading' : 'Send'}
        </button>
      </div>

      {/* PostBoy Info Bar (Stage 2: 80ms) */}
      <div
        className={`px-3 py-1 bg-[#1E1E1E] flex items-center justify-between border-b border-[#2D2D30] text-[11px] transition-all duration-300 ease-out delay-75 ${
          isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-2'
        }`}
      >
        <div className="flex items-center gap-2">
          <span className="text-white font-semibold">Create issue</span>
          <span className={`transition-colors duration-200 ${statusIsErr ? 'text-[#F48771]' : 'text-[#4EC9B0]'}`}>
            {statusMsg}
          </span>
        </div>
        <div className="flex items-center gap-1.5 select-none">
          <span className="px-2 py-0.2 rounded bg-[#007ACC] text-white font-bold text-[10px]">
            ● Production
          </span>
        </div>
      </div>

      {/* Dual Pane Workspace */}
      <div className="grid grid-cols-1 md:grid-cols-2 divide-y md:divide-y-0 md:divide-x divide-[#3E3E42] bg-[#1E1E1E] min-h-[360px]">
        
        {/* Left Pane: Request (Stage 2: 120ms) */}
        <div
          className={`flex flex-col bg-[#1E1E1E] transition-all duration-300 ease-out delay-100 ${
            isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-2'
          }`}
        >
          {/* Request Tab Bar */}
          <div className="flex border-b border-[#2D2D30] bg-[#252526]/50">
            {(['params', 'auth', 'headers', 'body'] as ReqTab[]).map((tab) => (
              <button
                key={tab}
                onClick={() => setReqTab(tab)}
                type="button"
                className={`px-3 py-1.5 text-[11px] font-semibold uppercase tracking-wider transition-all duration-150 ease-out border-b-2 hover:-translate-y-px active:translate-y-0 focus:outline-none focus-visible:ring-1 focus-visible:ring-[#FF6C37] ${
                  reqTab === tab
                    ? 'text-[#FF6C37] border-[#FF6C37] bg-[#1E1E1E]'
                    : 'text-[#858585] border-transparent hover:text-white hover:bg-[#252526]/80'
                }`}
              >
                {tab}
              </button>
            ))}
          </div>

          {/* Request Panel Content */}
          <div className="p-3 flex-1 bg-[#252526]/30 overflow-auto">
            {reqTab === 'body' && (
              <div className="space-y-2">
                <div className="flex items-center justify-between text-[10px] text-[#858585] pb-1 border-b border-[#2D2D30]">
                  <div className="flex gap-2">
                    <span className="text-[#FF6C37] font-semibold underline underline-offset-2">Raw (JSON)</span>
                    <span className="hover:text-white cursor-pointer transition-colors">Form URL-Encoded</span>
                    <span className="hover:text-white cursor-pointer transition-colors">Multipart</span>
                    <span className="hover:text-white cursor-pointer transition-colors">Binary</span>
                  </div>
                  <span className="text-[#6E6E6E] select-none">Alt+T cycles type</span>
                </div>
                
                <pre className="text-[#CCCCCC] leading-relaxed overflow-x-auto">
                  <span>{`{\n`}</span>
                  <span>  </span><span className="text-[#9CDCFE]">&quot;title&quot;</span><span>: </span><span className="text-[#CE9178]">&quot;CLI headless test runner fails in CI&quot;</span><span>{`,\n`}</span>
                  <span>  </span><span className="text-[#9CDCFE]">&quot;body&quot;</span><span>: </span><span className="text-[#CE9178]">&quot;Assertion expected_status matched, captures verified.&quot;</span><span>{`,\n`}</span>
                  <span>  </span><span className="text-[#9CDCFE]">&quot;labels&quot;</span><span>: [</span><span className="text-[#CE9178]">&quot;bug&quot;</span><span>, </span><span className="text-[#CE9178]">&quot;terminal&quot;</span><span>]{`,\n`}</span>
                  <span>  </span><span className="text-[#9CDCFE]">&quot;milestone&quot;</span><span>: </span><span className="text-[#B5CEA8]">1</span><span>{`,\n`}</span>
                  <span>  </span><span className="text-[#9CDCFE]">&quot;assignee&quot;</span><span>: </span><span className="text-[#CE9178]">&quot;&#123;&#123; ASSIGNEE &#125;&#125;&quot;</span><span>{`\n`}</span>
                  <span>{`}`}</span>
                </pre>

                <div className="pt-2 text-[10px] text-[#6E6E6E] flex justify-between border-t border-[#2D2D30]">
                  <span>Ctrl+O: edit body in $EDITOR</span>
                  <span className="text-[#9CDCFE]">Interpolated at send time</span>
                </div>
              </div>
            )}

            {reqTab === 'auth' && (
              <div className="space-y-3">
                <div className="text-[11px] text-[#858585] flex items-center gap-2">
                  <span>Type:</span>
                  <span className="text-[#FF6C37] font-bold">Bearer Token</span>
                  <span className="text-[#6E6E6E]">(←/→ cycle: Basic, API Key, None)</span>
                </div>
                <div className="space-y-1">
                  <div className="text-[#858585] text-[10px]">Token value:</div>
                  <div className="bg-[#1E1E1E] p-2 rounded border border-[#3E3E42] text-[#9CDCFE] flex items-center justify-between">
                    <span>&#123;&#123; secret.GITHUB_PAT &#125;&#125;</span>
                    <span className="text-[10px] text-[#4EC9B0] bg-[#4EC9B0]/10 px-1.5 py-0.5 rounded">
                      ● OS Keychain
                    </span>
                  </div>
                </div>
                <div className="text-[10px] text-[#858585] leading-relaxed">
                  PostBoy stores the raw token safely inside your system keyring. The plain-text value never touches <code className="text-[#9CDCFE]">config.json</code> or SQLite history.
                </div>
              </div>
            )}

            {reqTab === 'headers' && (
              <div className="space-y-1.5">
                <div className="text-[10px] text-[#858585] mb-2">n: new • Enter: edit • t: toggle • d: delete</div>
                {[
                  { key: 'Accept', val: 'application/vnd.github.v3+json', on: true },
                  { key: 'Content-Type', val: 'application/json', on: true },
                  { key: 'User-Agent', val: 'PostBoy/1.0', on: true },
                  { key: 'X-Debug', val: 'true', on: false },
                ].map((h, i) => (
                  <div key={i} className="flex items-center gap-2 text-[11px] py-0.5">
                    <span className={h.on ? 'text-[#4EC9B0]' : 'text-[#6E6E6E]'}>{h.on ? '✓' : '✗'}</span>
                    <span className="text-[#9CDCFE] font-semibold w-28">{h.key}:</span>
                    <span className={`flex-1 truncate ${h.on ? 'text-[#CCCCCC]' : 'text-[#6E6E6E] line-through'}`}>{h.val}</span>
                  </div>
                ))}
              </div>
            )}

            {reqTab === 'params' && (
              <div className="space-y-1.5">
                <div className="text-[10px] text-[#858585] mb-2">Query Parameters (appended at send time):</div>
                {[
                  { key: 'state', val: 'open', on: true },
                  { key: 'per_page', val: '25', on: true },
                  { key: 'sort', val: 'updated', on: true },
                ].map((p, i) => (
                  <div key={i} className="flex items-center gap-2 text-[11px] py-0.5">
                    <span className="text-[#4EC9B0]">✓</span>
                    <span className="text-[#9CDCFE] font-semibold w-24">{p.key}</span>
                    <span className="text-[#858585]">=</span>
                    <span className="text-[#CCCCCC]">{p.val}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Right Pane: Response (Stage 3 & 4: 180ms - 260ms) */}
        <div
          className={`flex flex-col bg-[#1E1E1E] transition-all duration-300 ease-out delay-150 ${
            isInView ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-2'
          }`}
        >
          {/* Response Tab Bar & Toolbar */}
          <div className="flex items-center justify-between border-b border-[#2D2D30] bg-[#252526]/50 pr-2">
            <div className="flex">
              {(['body', 'headers'] as ResTab[]).map((tab) => (
                <button
                  key={tab}
                  onClick={() => setResTab(tab)}
                  type="button"
                  className={`px-3 py-1.5 text-[11px] font-semibold uppercase tracking-wider transition-all duration-150 ease-out border-b-2 hover:-translate-y-px active:translate-y-0 focus:outline-none focus-visible:ring-1 focus-visible:ring-[#FF6C37] ${
                    resTab === tab
                      ? 'text-[#FF6C37] border-[#FF6C37] bg-[#1E1E1E]'
                      : 'text-[#858585] border-transparent hover:text-white hover:bg-[#252526]/80'
                  }`}
                >
                  {tab}
                </button>
              ))}
            </div>

            {/* Quick response tool hints */}
            <div className="hidden sm:flex items-center gap-2 text-[10px] text-[#858585]">
              <button
                type="button"
                onClick={() => handleFilterToggle('data.id')}
                className={`px-1.5 py-0.5 rounded border transition-all duration-150 hover:-translate-y-px active:translate-y-0 focus:outline-none focus-visible:ring-1 focus-visible:ring-[#FF6C37] ${
                  activeFilter === 'data.id'
                    ? 'border-[#FF6C37] text-[#FF6C37] bg-[#FF6C37]/10'
                    : 'border-[#3E3E42] text-[#858585] hover:text-[#CCCCCC] hover:border-[#505054]'
                }`}
                title="Filter JSON with gjson path"
              >
                f: filter
              </button>
              <button
                type="button"
                onClick={() => setIsRawMode(!isRawMode)}
                className={`px-1.5 py-0.5 rounded border transition-all duration-150 hover:-translate-y-px active:translate-y-0 focus:outline-none focus-visible:ring-1 focus-visible:ring-[#4EC9B0] ${
                  isRawMode
                    ? 'border-[#4EC9B0] text-[#4EC9B0] bg-[#4EC9B0]/10'
                    : 'border-[#3E3E42] text-[#858585] hover:text-[#CCCCCC] hover:border-[#505054]'
                }`}
              >
                r: {isRawMode ? 'raw' : 'pretty'}
              </button>
              <span className="text-[#6E6E6E] select-none">y copy • s save</span>
            </div>
          </div>

          {/* Response Meta Header (Subtle status reveal) */}
          <div className="px-3 py-1.5 bg-[#252526]/80 flex items-center justify-between border-b border-[#2D2D30] text-[11px]">
            <div className="flex items-center gap-3">
              <span className="text-[#4EC9B0] font-bold">201 Created</span>
              <span className="text-[#858585]">142ms</span>
              <span className="text-[#858585]">1.4 KB</span>
            </div>
            {isFilterActive && (
              <span className="text-[#DCDCAA] text-[10px]">
                filter: {activeFilter}
              </span>
            )}
          </div>

          {/* Response Content (JSON body) */}
          <div className="p-3 flex-1 bg-[#1E1E1E] overflow-auto">
            {resTab === 'body' && (
              <pre className="text-[#CCCCCC] leading-relaxed text-[11px] overflow-x-auto">
                {isFilterActive ? (
                  `84019284`
                ) : isRawMode ? (
                  `{"id":84019284,"node_id":"I_kwDOAG7x4858","number":84,"title":"CLI headless test runner fails in CI","user":{"login":"kambledheerajkumar","id":491823},"state":"open","created_at":"2026-10-08T02:14:00Z"}`
                ) : (
                  <>
                    <span>{`{\n`}</span>
                    <span>  </span><span className="text-[#9CDCFE]">&quot;id&quot;</span><span>: </span><span className="text-[#B5CEA8]">84019284</span><span>{`,\n`}</span>
                    <span>  </span><span className="text-[#9CDCFE]">&quot;node_id&quot;</span><span>: </span><span className="text-[#CE9178]">&quot;I_kwDOAG7x4858&quot;</span><span>{`,\n`}</span>
                    <span>  </span><span className="text-[#9CDCFE]">&quot;number&quot;</span><span>: </span><span className="text-[#B5CEA8]">84</span><span>{`,\n`}</span>
                    <span>  </span><span className="text-[#9CDCFE]">&quot;title&quot;</span><span>: </span><span className="text-[#CE9178]">&quot;CLI headless test runner fails in CI&quot;</span><span>{`,\n`}</span>
                    <span>  </span><span className="text-[#9CDCFE]">&quot;state&quot;</span><span>: </span><span className="text-[#4EC9B0]">&quot;open&quot;</span><span>{`,\n`}</span>
                    <span>  </span><span className="text-[#9CDCFE]">&quot;user&quot;</span><span>: {`{\n`}</span>
                    <span>    </span><span className="text-[#9CDCFE]">&quot;login&quot;</span><span>: </span><span className="text-[#CE9178]">&quot;kambledheerajkumar&quot;</span><span>{`,\n`}</span>
                    <span>    </span><span className="text-[#9CDCFE]">&quot;id&quot;</span><span>: </span><span className="text-[#B5CEA8]">491823</span><span>{`\n`}</span>
                    <span>  </span><span>{`},\n`}</span>
                    <span>  </span><span className="text-[#9CDCFE]">&quot;created_at&quot;</span><span>: </span><span className="text-[#CE9178]">&quot;2026-10-08T02:14:00Z&quot;</span><span>{`\n`}</span>
                    <span>{`}`}</span>
                  </>
                )}
              </pre>
            )}

            {resTab === 'headers' && (
              <div className="space-y-1 text-[11px]">
                {[
                  { k: 'content-type', v: 'application/json; charset=utf-8' },
                  { k: 'server', v: 'GitHub.com' },
                  { k: 'status', v: '201 Created' },
                  { k: 'cache-control', v: 'max-age=0, private, must-revalidate' },
                  { k: 'x-ratelimit-remaining', v: '4982' },
                  { k: 'x-ratelimit-reset', v: '1791458040' },
                ].map((header, idx) => (
                  <div key={idx} className="flex gap-2">
                    <span className="text-[#9CDCFE]">{header.k}:</span>
                    <span className="text-[#CCCCCC]">{header.v}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* PostBoy Footer Keymap Bar */}
      <div className="px-3 py-1.5 bg-[#252526] border-t border-[#3E3E42] flex flex-wrap items-center gap-x-4 gap-y-1 text-[10px] text-[#858585] overflow-x-auto select-none">
        <span><kbd className="text-[#FF6C37] font-bold">Alt+R</kbd> Send</span>
        <span><kbd className="text-[#FF6C37] font-bold">Ctrl+S</kbd> Save</span>
        <span><kbd className="text-[#FF6C37] font-bold">Alt+O</kbd> Collections</span>
        <span><kbd className="text-[#FF6C37] font-bold">Alt+N</kbd> New</span>
        <span><kbd className="text-[#FF6C37] font-bold">Alt+I</kbd> Import</span>
        <span><kbd className="text-[#FF6C37] font-bold">Alt+C</kbd> Copy curl</span>
        <span><kbd className="text-[#FF6C37] font-bold">Alt+H</kbd> History</span>
        <span><kbd className="text-[#FF6C37] font-bold">Alt+V</kbd> Env</span>
        <span><kbd className="text-[#FF6C37] font-bold">?</kbd> Help</span>
        <span><kbd className="text-[#FF6C37] font-bold">Ctrl+C</kbd> Quit</span>
      </div>
    </div>
  );
}
