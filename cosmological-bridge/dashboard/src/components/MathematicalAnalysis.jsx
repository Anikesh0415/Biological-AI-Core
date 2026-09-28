import React from 'react';
import { ScatterChart, Scatter, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { X } from 'lucide-react';

const CustomTooltip = ({ active, payload }) => {
  if (active && payload && payload.length) {
    const data = payload[0].payload;
    return (
      <div className="bg-black/90 backdrop-blur-xl border border-white/10 p-3 rounded-lg shadow-2xl">
        <p className="text-slate-300 text-[11px] uppercase tracking-wider font-semibold mb-2">Degree Point</p>
        <p className="text-slate-100 font-mono text-sm">
          <span className="text-slate-500 mr-2">k:</span>{data.k}
        </p>
        <p className="text-slate-100 font-mono text-sm">
          <span className="text-slate-500 mr-2">P(k):</span>{data.p_k.toFixed(4)}
        </p>
      </div>
    );
  }
  return null;
};

const MathematicalAnalysis = ({ bioData, cosmicData, onClose }) => {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-8">
      {/* Dimmed Background Overlay */}
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose}></div>
      
      {/* Modal Container */}
      <div className="relative w-full max-w-5xl h-[70vh] bg-[#050505] border border-white/10 rounded-2xl shadow-2xl flex flex-col overflow-hidden animate-in fade-in zoom-in-95 duration-200">
        
        {/* Header */}
        <div className="flex items-center justify-between px-8 py-5 border-b border-white/5 bg-white/[0.02]">
          <div>
            <h2 className="text-[15px] font-medium tracking-tight text-slate-50 mb-1">
              Degree Distribution: P(k)
            </h2>
            <p className="text-slate-400 text-[11px] font-medium leading-relaxed max-w-2xl">
              Log-Log plot testing for power-law relations P(k) ∝ k^(-γ). A straight downward slope on a logarithmic scale indicates a scale-free network topology.
            </p>
          </div>
          <button 
            onClick={onClose}
            className="p-2 rounded-full hover:bg-white/10 text-slate-400 hover:text-white transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Chart Area */}
        <div className="flex-1 w-full p-8 min-h-0">
          <ResponsiveContainer width="100%" height="100%">
            <ScatterChart margin={{ top: 20, right: 30, bottom: 20, left: 20 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" vertical={false} />
              
              <XAxis 
                type="number" 
                dataKey="k" 
                name="Degree (k)" 
                scale="log" 
                domain={['auto', 'auto']} 
                tickFormatter={(tick) => tick}
                stroke="rgba(255,255,255,0.2)"
                tick={{ fill: 'rgba(255,255,255,0.5)', fontSize: 11, fontFamily: "'JetBrains Mono', monospace" }}
                label={{ value: 'Degree (k) [Log Scale]', position: 'bottom', offset: 0, fill: 'rgba(255,255,255,0.5)', fontSize: 11 }}
              />
              
              <YAxis 
                type="number" 
                dataKey="p_k" 
                name="Probability P(k)" 
                scale="log" 
                domain={['auto', 'auto']} 
                tickFormatter={(tick) => tick.toExponential(1)}
                stroke="rgba(255,255,255,0.2)"
                tick={{ fill: 'rgba(255,255,255,0.5)', fontSize: 11, fontFamily: "'JetBrains Mono', monospace" }}
                label={{ value: 'Probability P(k) [Log Scale]', angle: -90, position: 'insideLeft', fill: 'rgba(255,255,255,0.5)', fontSize: 11 }}
              />
              
              <Tooltip content={<CustomTooltip />} cursor={{ strokeDasharray: '3 3', stroke: 'rgba(255,255,255,0.2)' }} />
              
              <Legend 
                wrapperStyle={{ paddingTop: '20px', fontSize: '12px' }}
                iconType="circle"
              />
              
              <Scatter 
                name="Biological Neural Network" 
                data={bioData} 
                fill="#3b82f6" // blue-500
                shape="circle"
                line={{ stroke: '#3b82f6', strokeWidth: 1.5, opacity: 0.8 }}
              />
              <Scatter 
                name="Cosmic Web (SDSS)" 
                data={cosmicData} 
                fill="#8b5cf6" // violet-500
                shape="circle"
                line={{ stroke: '#8b5cf6', strokeWidth: 1.5, opacity: 0.8 }}
              />
            </ScatterChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  );
};

export default MathematicalAnalysis;
