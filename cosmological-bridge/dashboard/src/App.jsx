import React, { useState } from 'react';
import NetworkPanel from './components/NetworkPanel';
import MathematicalAnalysis from './components/MathematicalAnalysis';
import bioData from './data/neural_network_data.json';
import cosmicData from './data/cosmic_network_data.json';
import bioAdvanced from './data/neural_advanced_metrics.json';
import cosmicAdvanced from './data/cosmic_advanced_metrics.json';
import { Layers, Activity } from 'lucide-react';

function App() {
  const [isHeatmap, setIsHeatmap] = useState(false);
  const [showChart, setShowChart] = useState(false);

  return (
    <div className="h-screen w-screen overflow-hidden bg-black flex flex-col relative" style={{ fontFamily: "'Inter', sans-serif" }}>
      
      {/* Floating Top Controls (HUD) */}
      <div className="absolute top-6 left-1/2 -translate-x-1/2 z-30 flex items-center space-x-4 bg-black/40 backdrop-blur-md border border-white/10 px-5 py-2.5 rounded-full shadow-2xl">
        <div className="flex items-center space-x-2 mr-4 border-r border-white/10 pr-4">
          <Layers className="w-4 h-4 text-white" />
          <span className="text-xs font-semibold tracking-wide text-white uppercase">Isomorphism Engine</span>
        </div>
        
        {/* Heatmap Toggle */}
        <div className="flex items-center space-x-3">
          <span className={`text-[10px] font-bold uppercase tracking-wider ${!isHeatmap ? 'text-white' : 'text-slate-500'}`}>Topology</span>
          <button 
            onClick={() => setIsHeatmap(!isHeatmap)}
            className={`relative inline-flex h-4 w-8 items-center rounded-full transition-colors duration-200 focus:outline-none ${isHeatmap ? 'bg-orange-600' : 'bg-slate-700'}`}
          >
            <span className={`inline-block h-2.5 w-2.5 transform rounded-full bg-white shadow-sm transition duration-200 ${isHeatmap ? 'translate-x-4' : 'translate-x-1'}`} />
          </button>
          <span className={`text-[10px] font-bold uppercase tracking-wider ${isHeatmap ? 'text-orange-400' : 'text-slate-500'}`}>PageRank</span>
        </div>
      </div>

      {/* Main Split Screen for Visualizers (100% Height edge-to-edge) */}
      <main className="flex w-full h-full relative">
        {/* Absolute Center Divider */}
        <div className="absolute inset-y-0 left-1/2 w-px bg-white/10 z-20 -translate-x-1/2"></div>
        
        {/* Left Panel: Biology */}
        <div className="w-1/2 h-full relative">
          <NetworkPanel 
            title="Biological Neural Network" 
            data={bioData} 
            type="bio"
            colorHex="#3b82f6" // blue-500
            advancedMetrics={bioAdvanced}
            isHeatmap={isHeatmap}
          />
        </div>

        {/* Right Panel: Cosmology */}
        <div className="w-1/2 h-full relative">
          <NetworkPanel 
            title="Cosmic Web (SDSS)" 
            data={cosmicData} 
            type="cosmic"
            colorHex="#8b5cf6" // violet-500
            advancedMetrics={cosmicAdvanced}
            isHeatmap={isHeatmap}
          />
        </div>
      </main>

      {/* Floating Action Button for Chart */}
      <button 
        onClick={() => setShowChart(true)}
        className="absolute bottom-16 left-1/2 -translate-x-1/2 z-30 flex items-center space-x-2 bg-white text-black px-5 py-2.5 rounded-full font-semibold text-xs shadow-[0_0_20px_rgba(255,255,255,0.2)] hover:bg-slate-200 transition-colors"
      >
        <Activity className="w-4 h-4" />
        <span>View Mathematical Analysis</span>
      </button>

      {/* Modal for Mathematical Analysis */}
      {showChart && (
        <MathematicalAnalysis 
          bioData={bioAdvanced.degree_distribution} 
          cosmicData={cosmicAdvanced.degree_distribution} 
          onClose={() => setShowChart(false)}
        />
      )}

      {/* HUD Bottom Bar / Footer */}
      <footer className="absolute bottom-0 w-full bg-black/40 backdrop-blur-md border-t border-white/10 px-8 py-3 flex items-center justify-between z-30 pointer-events-none">
        <div className="text-[10px] font-mono text-white/50 tracking-wider">
          PROJECT: THE COSMOLOGICAL BRIDGE — ENGINEERED BY ANIKESH
        </div>
        <div className="text-[10px] font-mono text-white/50 tracking-wider">
          DATA PROVIDERS: <span className="text-blue-400/80">FINALSPARK (NEUROBIOLOGICAL MEA)</span> | <span className="text-violet-400/80">SLOAN DIGITAL SKY SURVEY (COSMOLOGICAL SDSS)</span>
        </div>
      </footer>
    </div>
  );
}

export default App;
