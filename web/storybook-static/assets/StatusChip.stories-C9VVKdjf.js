import{n as e}from"./chunk-BneVvdWh.js";import{d as t,f as n,t as r,v as i}from"./iframe-DhV8BIVS.js";function a({status:e,size:t=`small`}){return(0,o.jsx)(n,{label:e,color:s[e]??`default`,size:t,variant:e===`running`?`filled`:`outlined`})}var o,s,c=e((()=>{r(),o=i(),s={pending:`default`,ready:`info`,running:`info`,succeeded:`success`,failed:`error`,canceled:`warning`},a.__docgenInfo={description:``,methods:[],displayName:`StatusChip`,props:{status:{required:!0,tsType:{name:`union`,raw:`WorkflowStatus | OpStatus`,elements:[{name:`union`,raw:`'pending' | 'running' | 'succeeded' | 'failed' | 'canceled'`,elements:[{name:`literal`,value:`'pending'`},{name:`literal`,value:`'running'`},{name:`literal`,value:`'succeeded'`},{name:`literal`,value:`'failed'`},{name:`literal`,value:`'canceled'`}]},{name:`union`,raw:`'pending' | 'ready' | 'running' | 'succeeded' | 'failed' | 'canceled'`,elements:[{name:`literal`,value:`'pending'`},{name:`literal`,value:`'ready'`},{name:`literal`,value:`'running'`},{name:`literal`,value:`'succeeded'`},{name:`literal`,value:`'failed'`},{name:`literal`,value:`'canceled'`}]}]},description:``},size:{required:!1,tsType:{name:`union`,raw:`'small' | 'medium'`,elements:[{name:`literal`,value:`'small'`},{name:`literal`,value:`'medium'`}]},description:``,defaultValue:{value:`'small'`,computed:!1}}}}})),l,u,d,f,p,m,h,g,_,v;e((()=>{c(),r(),l=i(),u={title:`Common/StatusChip`,component:a},d={args:{status:`pending`}},f={args:{status:`ready`}},p={args:{status:`running`}},m={args:{status:`succeeded`}},h={args:{status:`failed`}},g={args:{status:`canceled`}},_={render:()=>(0,l.jsxs)(t,{sx:{display:`flex`,gap:1},children:[(0,l.jsx)(a,{status:`pending`}),(0,l.jsx)(a,{status:`ready`}),(0,l.jsx)(a,{status:`running`}),(0,l.jsx)(a,{status:`succeeded`}),(0,l.jsx)(a,{status:`failed`}),(0,l.jsx)(a,{status:`canceled`})]})},d.parameters={...d.parameters,docs:{...d.parameters?.docs,source:{originalSource:`{
  args: {
    status: 'pending'
  }
}`,...d.parameters?.docs?.source}}},f.parameters={...f.parameters,docs:{...f.parameters?.docs,source:{originalSource:`{
  args: {
    status: 'ready'
  }
}`,...f.parameters?.docs?.source}}},p.parameters={...p.parameters,docs:{...p.parameters?.docs,source:{originalSource:`{
  args: {
    status: 'running'
  }
}`,...p.parameters?.docs?.source}}},m.parameters={...m.parameters,docs:{...m.parameters?.docs,source:{originalSource:`{
  args: {
    status: 'succeeded'
  }
}`,...m.parameters?.docs?.source}}},h.parameters={...h.parameters,docs:{...h.parameters?.docs,source:{originalSource:`{
  args: {
    status: 'failed'
  }
}`,...h.parameters?.docs?.source}}},g.parameters={...g.parameters,docs:{...g.parameters?.docs,source:{originalSource:`{
  args: {
    status: 'canceled'
  }
}`,...g.parameters?.docs?.source}}},_.parameters={..._.parameters,docs:{..._.parameters?.docs,source:{originalSource:`{
  render: () => <Box sx={{
    display: 'flex',
    gap: 1
  }}>
      <StatusChip status="pending" />
      <StatusChip status="ready" />
      <StatusChip status="running" />
      <StatusChip status="succeeded" />
      <StatusChip status="failed" />
      <StatusChip status="canceled" />
    </Box>
}`,..._.parameters?.docs?.source}}},v=[`Pending`,`Ready`,`Running`,`Succeeded`,`Failed`,`Canceled`,`AllStatuses`]}))();export{_ as AllStatuses,g as Canceled,h as Failed,d as Pending,f as Ready,p as Running,m as Succeeded,v as __namedExportsOrder,u as default};