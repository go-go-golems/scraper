import{n as e}from"./chunk-BneVvdWh.js";import{d as t,l as n,m as r,s as i,t as a,u as o,v as s}from"./iframe-DhV8BIVS.js";import{i as c,r as l}from"./factories-DSvIXpWH.js";function u({queues:e,maxVisible:a=6}){let s=e.slice(0,a);return(0,d.jsx)(o,{children:(0,d.jsxs)(n,{children:[(0,d.jsx)(r,{variant:`body2`,color:`text.secondary`,gutterBottom:!0,children:`Queue Health`}),s.length===0&&(0,d.jsx)(r,{variant:`body2`,color:`text.disabled`,sx:{mt:1},children:`No active queues`}),(0,d.jsx)(t,{sx:{display:`flex`,flexDirection:`column`,gap:1.5,mt:1},children:s.map(e=>{let n=e.maxInFlight>0?e.inFlight/e.maxInFlight*100:0,a=n>=90?`error`:n>=50?`warning`:`primary`;return(0,d.jsxs)(t,{children:[(0,d.jsxs)(t,{sx:{display:`flex`,justifyContent:`space-between`,mb:.5},children:[(0,d.jsx)(r,{variant:`caption`,noWrap:!0,sx:{maxWidth:`70%`},children:e.queue}),(0,d.jsxs)(r,{variant:`caption`,color:`text.secondary`,children:[e.inFlight,`/`,e.maxInFlight]})]}),(0,d.jsx)(i,{variant:`determinate`,value:Math.min(n,100),color:a,sx:{height:6,borderRadius:1}})]},`${e.site}:${e.queue}`)})})]})})}var d,f=e((()=>{a(),d=s(),u.__docgenInfo={description:``,methods:[],displayName:`QueueHealthPreview`,props:{queues:{required:!0,tsType:{name:`Array`,elements:[{name:`QueueStatus`}],raw:`QueueStatus[]`},description:``},maxVisible:{required:!1,tsType:{name:`number`},description:``,defaultValue:{value:`6`,computed:!1}}}}})),p,m,h,g,_,v;e((()=>{f(),c(),p={title:`Overview/QueueHealthPreview`,component:u},m={args:{queues:[l({queue:`site:hn:http`,inFlight:2,maxInFlight:4}),l({queue:`site:hn:js`,inFlight:1,maxInFlight:4}),l({site:`slashdot`,queue:`site:sd:http`,inFlight:0,maxInFlight:4}),l({site:`nereval`,queue:`site:nv:js`,inFlight:0,maxInFlight:1})]}},h={args:{queues:[l({queue:`site:hn:http`,inFlight:4,maxInFlight:4}),l({queue:`site:hn:js`,inFlight:4,maxInFlight:4}),l({site:`nereval`,queue:`site:nv:http`,inFlight:4,maxInFlight:4})]}},g={args:{queues:[l({queue:`site:hn:http`,inFlight:0,maxInFlight:4}),l({queue:`site:hn:js`,inFlight:0,maxInFlight:4})]}},_={args:{queues:[]}},m.parameters={...m.parameters,docs:{...m.parameters?.docs,source:{originalSource:`{
  args: {
    queues: [createQueueStatus({
      queue: 'site:hn:http',
      inFlight: 2,
      maxInFlight: 4
    }), createQueueStatus({
      queue: 'site:hn:js',
      inFlight: 1,
      maxInFlight: 4
    }), createQueueStatus({
      site: 'slashdot',
      queue: 'site:sd:http',
      inFlight: 0,
      maxInFlight: 4
    }), createQueueStatus({
      site: 'nereval',
      queue: 'site:nv:js',
      inFlight: 0,
      maxInFlight: 1
    })]
  }
}`,...m.parameters?.docs?.source}}},h.parameters={...h.parameters,docs:{...h.parameters?.docs,source:{originalSource:`{
  args: {
    queues: [createQueueStatus({
      queue: 'site:hn:http',
      inFlight: 4,
      maxInFlight: 4
    }), createQueueStatus({
      queue: 'site:hn:js',
      inFlight: 4,
      maxInFlight: 4
    }), createQueueStatus({
      site: 'nereval',
      queue: 'site:nv:http',
      inFlight: 4,
      maxInFlight: 4
    })]
  }
}`,...h.parameters?.docs?.source}}},g.parameters={...g.parameters,docs:{...g.parameters?.docs,source:{originalSource:`{
  args: {
    queues: [createQueueStatus({
      queue: 'site:hn:http',
      inFlight: 0,
      maxInFlight: 4
    }), createQueueStatus({
      queue: 'site:hn:js',
      inFlight: 0,
      maxInFlight: 4
    })]
  }
}`,...g.parameters?.docs?.source}}},_.parameters={..._.parameters,docs:{..._.parameters?.docs,source:{originalSource:`{
  args: {
    queues: []
  }
}`,..._.parameters?.docs?.source}}},v=[`Default`,`Saturated`,`AllIdle`,`Empty`]}))();export{g as AllIdle,m as Default,_ as Empty,h as Saturated,v as __namedExportsOrder,p as default};