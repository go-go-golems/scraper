import{n as e}from"./chunk-BneVvdWh.js";import{a as t,d as n,l as r,m as i,t as a,u as o,v as s}from"./iframe-DhV8BIVS.js";function c({counts:e}){let a=Object.values(e).reduce((e,t)=>e+(t??0),0);return a===0?null:(0,l.jsx)(o,{children:(0,l.jsxs)(r,{children:[(0,l.jsx)(i,{variant:`body2`,color:`text.secondary`,gutterBottom:!0,children:`Op Status Breakdown`}),(0,l.jsx)(n,{sx:{display:`flex`,height:24,borderRadius:1,overflow:`hidden`,mt:1},children:u.map(({key:r,label:i,color:o})=>{let s=e[r]??0;if(s===0)return null;let c=s/a*100;return(0,l.jsx)(t,{title:`${i}: ${s} (${c.toFixed(1)}%)`,children:(0,l.jsx)(n,{sx:{width:`${c}%`,bgcolor:o,minWidth:s>0?4:0}})},r)})}),(0,l.jsx)(n,{sx:{display:`flex`,gap:2,mt:1.5,flexWrap:`wrap`},children:u.map(({key:t,label:r,color:a})=>{let o=e[t]??0;return o===0?null:(0,l.jsxs)(n,{sx:{display:`flex`,alignItems:`center`,gap:.5},children:[(0,l.jsx)(n,{sx:{width:10,height:10,borderRadius:`50%`,bgcolor:a}}),(0,l.jsxs)(i,{variant:`caption`,children:[r,`: `,o]})]},t)})})]})})}var l,u,d=e((()=>{a(),l=s(),u=[{key:`pending`,label:`Pending`,color:`#bdbdbd`},{key:`ready`,label:`Ready`,color:`#42a5f5`},{key:`running`,label:`Running`,color:`#1976d2`},{key:`succeeded`,label:`Succeeded`,color:`#66bb6a`},{key:`failed`,label:`Failed`,color:`#ef5350`},{key:`canceled`,label:`Canceled`,color:`#ffa726`}],c.__docgenInfo={description:``,methods:[],displayName:`OpStatusBreakdown`,props:{counts:{required:!0,tsType:{name:`Partial`,elements:[{name:`Record`,elements:[{name:`union`,raw:`'pending' | 'ready' | 'running' | 'succeeded' | 'failed' | 'canceled'`,elements:[{name:`literal`,value:`'pending'`},{name:`literal`,value:`'ready'`},{name:`literal`,value:`'running'`},{name:`literal`,value:`'succeeded'`},{name:`literal`,value:`'failed'`},{name:`literal`,value:`'canceled'`}]},{name:`number`}],raw:`Record<OpStatus, number>`}],raw:`Partial<Record<OpStatus, number>>`},description:``}}}})),f,p,m,h,g,_,v;e((()=>{d(),f={title:`Overview/OpStatusBreakdown`,component:c},p={args:{counts:{pending:3,ready:23,running:4,succeeded:808,failed:12,canceled:0}}},m={args:{counts:{pending:0,ready:0,running:0,succeeded:500,failed:0,canceled:0}}},h={args:{counts:{pending:200,ready:15,running:3,succeeded:12,failed:0,canceled:0}}},g={args:{counts:{pending:0,ready:2,running:1,succeeded:40,failed:15,canceled:0}}},_={args:{counts:{pending:0,ready:0,running:1,succeeded:3,failed:0,canceled:0}}},p.parameters={...p.parameters,docs:{...p.parameters?.docs,source:{originalSource:`{
  args: {
    counts: {
      pending: 3,
      ready: 23,
      running: 4,
      succeeded: 808,
      failed: 12,
      canceled: 0
    }
  }
}`,...p.parameters?.docs?.source}}},m.parameters={...m.parameters,docs:{...m.parameters?.docs,source:{originalSource:`{
  args: {
    counts: {
      pending: 0,
      ready: 0,
      running: 0,
      succeeded: 500,
      failed: 0,
      canceled: 0
    }
  }
}`,...m.parameters?.docs?.source}}},h.parameters={...h.parameters,docs:{...h.parameters?.docs,source:{originalSource:`{
  args: {
    counts: {
      pending: 200,
      ready: 15,
      running: 3,
      succeeded: 12,
      failed: 0,
      canceled: 0
    }
  }
}`,...h.parameters?.docs?.source}}},g.parameters={...g.parameters,docs:{...g.parameters?.docs,source:{originalSource:`{
  args: {
    counts: {
      pending: 0,
      ready: 2,
      running: 1,
      succeeded: 40,
      failed: 15,
      canceled: 0
    }
  }
}`,...g.parameters?.docs?.source}}},_.parameters={..._.parameters,docs:{..._.parameters?.docs,source:{originalSource:`{
  args: {
    counts: {
      pending: 0,
      ready: 0,
      running: 1,
      succeeded: 3,
      failed: 0,
      canceled: 0
    }
  }
}`,..._.parameters?.docs?.source}}},v=[`Default`,`AllSucceeded`,`MostlyPending`,`HasFailures`,`SmallWorkflow`]}))();export{m as AllSucceeded,p as Default,g as HasFailures,h as MostlyPending,_ as SmallWorkflow,v as __namedExportsOrder,f as default};