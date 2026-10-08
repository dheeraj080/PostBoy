import React, { useState, useRef, useEffect } from 'react';
import { Copy, Check } from 'lucide-react';

interface CopyButtonProps {
  text: string;
  label?: string;
  className?: string;
}

export function CopyButton({ text, label, className = '' }: CopyButtonProps) {
  const [copied, setCopied] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (timeoutRef.current) clearTimeout(timeoutRef.current);
    };
  }, []);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(text);
    } catch {
      // Fallback
      const ta = document.createElement('textarea');
      ta.value = text;
      ta.style.position = 'fixed';
      ta.style.opacity = '0';
      document.body.appendChild(ta);
      ta.select();
      document.execCommand('copy');
      document.body.removeChild(ta);
    }

    if (timeoutRef.current) clearTimeout(timeoutRef.current);
    setCopied(true);
    timeoutRef.current = setTimeout(() => {
      setCopied(false);
    }, 1800);
  };

  return (
    <button
      onClick={handleCopy}
      type="button"
      aria-label={copied ? "Copied command to clipboard" : (label ? `Copy ${label}` : "Copy to clipboard")}
      aria-live="polite"
      className={`inline-flex items-center justify-center gap-1.5 px-2.5 py-1 text-xs font-mono rounded transition-all duration-150 ease-out hover:-translate-y-px active:translate-y-0 cursor-pointer select-none focus:outline-none focus-visible:ring-1 focus-visible:ring-[#FF6C37] min-w-[70px] ${
        copied
          ? 'bg-[#4EC9B0]/20 text-[#4EC9B0] border border-[#4EC9B0]/50 shadow-sm'
          : 'bg-[#1E1E1E] hover:bg-[#2e2e30] text-[#9CDCFE] hover:text-white border border-[#3E3E42] hover:border-[#505054]'
      } ${className}`}
    >
      <span className="shrink-0 transition-transform duration-150">
        {copied ? (
          <Check className="w-3.5 h-3.5 text-[#4EC9B0]" />
        ) : (
          <Copy className="w-3.5 h-3.5 opacity-80" />
        )}
      </span>
      <span className="font-medium tracking-tight">
        {copied ? 'Copied' : (label || 'Copy')}
      </span>
    </button>
  );
}
