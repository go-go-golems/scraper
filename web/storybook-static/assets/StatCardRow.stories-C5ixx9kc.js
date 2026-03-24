import{n as e}from"./chunk-BneVvdWh.js";import{c as t,t as n,v as r}from"./iframe-DhV8BIVS.js";import{i,n as a,t as o}from"./factories-DSvIXpWH.js";import{n as s,t as c}from"./StatCard-BKpmVkxn.js";function l({status:e,loading:n}){if(n||!e)return(0,u.jsx)(t,{container:!0,spacing:2,children:[0,1,2,3].map(e=>(0,u.jsx)(t,{size:{xs:12,sm:6,md:3},children:(0,u.jsx)(c,{title:``,value:0,loading:!0})},e))});let r=e.OpCounts,i=Object.values(r).reduce((e,t)=>e+t,0);return(0,u.jsxs)(t,{container:!0,spacing:2,children:[(0,u.jsx)(t,{size:{xs:12,sm:6,md:3},children:(0,u.jsx)(c,{title:`Workflows`,value:e.WorkflowCount,breakdown:[{label:`running`,value:r.running>0?Math.min(e.WorkflowCount,r.running):0,color:`info`}]})}),(0,u.jsx)(t,{size:{xs:12,sm:6,md:3},children:(0,u.jsx)(c,{title:`Operations`,value:i,breakdown:[{label:`ready`,value:r.ready??0,color:`info`},{label:`running`,value:r.running??0,color:`primary`},{label:`failed`,value:r.failed??0,color:`error`}]})}),(0,u.jsx)(t,{size:{xs:12,sm:6,md:3},children:(0,u.jsx)(c,{title:`Leases`,value:e.ActiveLeases,breakdown:[{label:`expired`,value:e.ExpiredLeases,color:e.ExpiredLeases>0?`warning`:`default`}]})}),(0,u.jsx)(t,{size:{xs:12,sm:6,md:3},children:(0,u.jsx)(c,{title:`Artifacts`,value:e.ArtifactCount})})]})}var u,d=e((()=>{n(),s(),u=r(),l.__docgenInfo={description:``,methods:[],displayName:`StatCardRow`,props:{status:{required:!1,tsType:{name:`EngineStatus`},description:``},loading:{required:!1,tsType:{name:`boolean`},description:``}}}})),f,p,m,h,g,_;e((()=>{d(),i(),f={title:`Overview/StatCardRow`,component:l},p={args:{status:a()}},m={args:{status:o()}},h={args:{loading:!0}},g={args:{status:a({WorkflowCount:87,OpCounts:{pending:120,ready:340,running:24,succeeded:12400,failed:89,canceled:3},ActiveLeases:24,ExpiredLeases:2,ArtifactCount:8900})}},p.parameters={...p.parameters,docs:{...p.parameters?.docs,source:{originalSource:`{
  args: {
    status: createEngineStatus()
  }
}`,...p.parameters?.docs?.source}}},m.parameters={...m.parameters,docs:{...m.parameters?.docs,source:{originalSource:`{
  args: {
    status: createEmptyEngineStatus()
  }
}`,...m.parameters?.docs?.source}}},h.parameters={...h.parameters,docs:{...h.parameters?.docs,source:{originalSource:`{
  args: {
    loading: true
  }
}`,...h.parameters?.docs?.source}}},g.parameters={...g.parameters,docs:{...g.parameters?.docs,source:{originalSource:`{
  args: {
    status: createEngineStatus({
      WorkflowCount: 87,
      OpCounts: {
        pending: 120,
        ready: 340,
        running: 24,
        succeeded: 12400,
        failed: 89,
        canceled: 3
      },
      ActiveLeases: 24,
      ExpiredLeases: 2,
      ArtifactCount: 8900
    })
  }
}`,...g.parameters?.docs?.source}}},_=[`Default`,`Empty`,`Loading`,`HighActivity`]}))();export{p as Default,m as Empty,g as HighActivity,h as Loading,_ as __namedExportsOrder,f as default};