/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import React from 'react';
import { Header } from './components/Header';
import { Hero } from './components/Hero';
import { WhyPostBoy } from './components/WhyPostBoy';
import { FeatureGrid } from './components/FeatureGrid';
import { Installation } from './components/Installation';
import { QuickStart } from './components/QuickStart';
import { HeadlessCI } from './components/HeadlessCI';
import { ImportExport } from './components/ImportExport';
import { Security } from './components/Security';
import { FAQ } from './components/FAQ';
import { Footer } from './components/Footer';

export default function App() {
  return (
    <div className="min-h-screen bg-[#1E1E1E] text-[#CCCCCC] flex flex-col font-sans selection:bg-[#FF6C37]/30 selection:text-white">
      {/* Skip to main content link for keyboard accessibility */}
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-50 focus:px-4 focus:py-2 focus:bg-[#FF6C37] focus:text-white focus:font-mono focus:text-xs focus:rounded shadow-lg"
      >
        Skip to main content
      </a>

      {/* 1. Header */}
      <Header />

      <main id="main-content" className="flex-1">
        {/* 2. Hero */}
        <Hero />

        {/* 3. Product proof / key benefits */}
        <WhyPostBoy />

        {/* 4. Feature grid */}
        <FeatureGrid />

        {/* 5. Installation */}
        <Installation />

        {/* 6. Quick start */}
        <QuickStart />

        {/* 7. Headless CI runner */}
        <HeadlessCI />

        {/* 8. Import/export */}
        <ImportExport />

        {/* 9. Security */}
        <Security />

        {/* 10. FAQ */}
        <FAQ />
      </main>

      {/* 11. Footer */}
      <Footer />
    </div>
  );
}
