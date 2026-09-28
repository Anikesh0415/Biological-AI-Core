import React, { useRef, useEffect, useState, useMemo } from 'react';
import ForceGraph3D from 'react-force-graph-3d';
import { Activity, Share2, Award, Zap } from 'lucide-react';

const NetworkPanel = ({ title, data, type, colorHex, advancedMetrics, isHeatmap }) => {
  const fgRef = useRef();
  const [containerSize, setContainerSize] = useState({ width: 0, height: 0 });
  const containerRef = useRef(null);

  useEffect(() => {
    const handleResize = () => {
      if (containerRef.current) {
        setContainerSize({
          width: containerRef.current.clientWidth,
          height: containerRef.current.clientHeight
        });
      }
    };

    handleResize();
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  const metricsMap = useMemo(() => {
    const map = new Map();
    if (advancedMetrics && advancedMetrics.node_metrics) {
      advancedMetrics.node_metrics.forEach(n => {
        map.set(n.id.toString(), n);
      });
    }
    return map;
  }, [advancedMetrics]);

  const topNodes = useMemo(() => {
    if (!advancedMetrics || !advancedMetrics.node_metrics) return [];
    return [...advancedMetrics.node_metrics]
      .sort((a, b) => b.pagerank - a.pagerank)
      .slice(0, 3);
  }, [advancedMetrics]);

  const getHeatmapColor = (pagerank) => {
    const maxPR = topNodes[0]?.pagerank || 1;
    const ratio = Math.min(1, Math.max(0, pagerank / maxPR));
    const r = Math.round(148 + (249 - 148) * ratio);
    const g = Math.round(163 + (115 - 163) * ratio);
    const b = Math.round(184 + (22 - 184) * ratio);
    const a = 0.7 + (0.3 * ratio);
    return `rgba(${r}, ${g}, ${b}, ${a})`;
  };

  const determineNodeColor = (node) => {
    if (!isHeatmap) return colorHex;
    const metrics = metricsMap.get(node.id.toString());
    if (!metrics) return 'rgba(71, 85, 105, 0.8)';
    return getHeatmapColor(metrics.pagerank);
  };

  const determineNodeVal = (node) => {
    // True native scaling
    const baseSize = type === 'cosmic' ? 0.1 : 1.5;
    
    if (!isHeatmap) return baseSize;
    
    const metrics = metricsMap.get(node.id.toString());
    if (!metrics) return baseSize;
    
    const maxPR = topNodes[0]?.pagerank || 1;
    const ratio = metrics.pagerank / maxPR;
    const scaleFactor = type === 'cosmic' ? 0.3 : 2.5; 
    return baseSize + (ratio * scaleFactor); 
  };

  const processedData = useMemo(() => {
    const nodes = data.nodes.map(n => {
      const newNode = { ...n };
      // 100% mathematically accurate coordinates. No spreading or distortion.
      if (newNode.x !== undefined && newNode.y !== undefined && newNode.z !== undefined) {
        newNode.fx = newNode.x;
        newNode.fy = newNode.y;
        newNode.fz = newNode.z;
      }
      return newNode;
    });
    return { nodes, links: data.edges };
  }, [data]);

  useEffect(() => {
    // Zoom to fit with normal padding
    const timer = setTimeout(() => {
      if (fgRef.current) {
        fgRef.current.zoomToFit(800, 50); 
      }
    }, 600);
    return () => clearTimeout(timer);
  }, [data, type]);

  return (
    <div className="relative w-full h-full flex flex-col" ref={containerRef}>
      {/* 3D Force Graph */}
      <div className="absolute inset-0 z-0 bg-transparent"> 
        {containerSize.width > 0 && (
          <ForceGraph3D
            ref={fgRef}
            width={containerSize.width}
            height={containerSize.height}
            graphData={processedData}
            nodeLabel="id"
            nodeColor={determineNodeColor}
            nodeResolution={16}
            linkColor={() => 'rgba(255,255,255,0.25)'} 
            linkWidth={type === 'cosmic' ? 0.2 : 1.0} 
            backgroundColor="rgba(0,0,0,0)"
            enableNodeDrag={false}
            enableNavigationControls={true}
            showNavInfo={false}
            nodeVal={determineNodeVal}
            cooldownTicks={type === 'cosmic' ? 0 : undefined} 
          />
        )}
      </div>

      {/* Minimal Info Overlay (Top Left) */}
      <div className="absolute top-8 left-8 z-10 min-w-[200px] pointer-events-auto">
        <h2 className="text-[15px] font-medium tracking-tight mb-1 text-white flex items-center">
          <span className="w-1.5 h-1.5 rounded-full mr-3 shadow-[0_0_8px_currentColor]" style={{ backgroundColor: colorHex, color: colorHex }}></span>
          {title}
        </h2>
        <p className="text-white/40 text-[9px] mb-6 uppercase tracking-[0.2em] font-semibold flex items-center">
          <Zap className="w-3 h-3 mr-1.5 opacity-50" />
          {data.metrics?.network_type || type}
        </p>
        
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <span className="text-[10px] font-medium text-white/50 uppercase tracking-widest">Nodes</span>
            <span className="font-mono text-[13px] text-white">{data.nodes.length}</span>
          </div>
          
          <div className="flex items-center justify-between">
            <span className="text-[10px] font-medium text-white/50 uppercase tracking-widest">Edges</span>
            <span className="font-mono text-[13px] text-white">{data.edges.length}</span>
          </div>

          <div className="h-[1px] w-full bg-gradient-to-r from-white/20 to-transparent my-4"></div>

          <div className="flex items-center justify-between">
            <span className="text-[10px] font-medium text-white/50 uppercase tracking-widest">Avg Degree</span>
            <span className="font-mono text-xs text-white">{data.metrics?.average_degree?.toFixed(2) || '0.00'}</span>
          </div>
          
          <div className="flex items-center justify-between">
            <span className="text-[10px] font-medium text-white/50 uppercase tracking-widest">Clustering</span>
            <span className="font-mono text-xs text-white">{data.metrics?.clustering_coefficient?.toFixed(4) || '0.0000'}</span>
          </div>
        </div>
      </div>

      {/* Leaderboard Overlay (Bottom Right) */}
      <div className="absolute bottom-20 right-8 z-10 min-w-[220px] pointer-events-auto">
        <div className="flex items-center mb-5 border-b border-white/10 pb-3">
          <Award className="w-3.5 h-3.5 text-white/40 mr-2" />
          <h3 className="text-[10px] font-semibold text-white/60 tracking-[0.2em] uppercase">Structural Hubs</h3>
        </div>
        <div className="space-y-4">
          {topNodes.map((n, idx) => (
            <div key={n.id} className="flex justify-between items-center text-xs">
              <div className="flex items-center">
                <span className="text-white/30 font-mono w-4 font-bold text-[10px]">0{idx + 1}.</span>
                <span className="text-white font-mono ml-2 truncate max-w-[80px]" title={n.id}>{n.id}</span>
              </div>
              <div className="text-right flex flex-row space-x-5 items-center">
                <div className="text-white/30 font-mono text-[10px]">k={n.degree}</div>
                <div className="text-orange-400 font-mono font-medium text-[11px] drop-shadow-[0_0_5px_rgba(251,146,60,0.5)]">{n.pagerank.toFixed(4)}</div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default NetworkPanel;
