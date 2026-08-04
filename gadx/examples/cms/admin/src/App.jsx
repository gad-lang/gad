const {useEffect, useState} = React;

const emptyPage = {title:"", slug:"", summary:"", body:"", coverImage:"", images:"", published:true};
const emptyTag = {name:"", slug:""};
const emptyPost = {title:"", slug:"", summary:"", body:"", rightBody:"", coverImage:"", images:"", published:true, tags:[]};
const emptyMenu = {label:"", kind:"page", pageId:null, tagId:null, url:"", position:10, visible:true};

function slugify(value){return (value||"").toLowerCase().trim().replace(/[^a-z0-9]+/g,"-").replace(/(^-|-$)/g,"");}
async function api(path, options={}){
  const res = await fetch(path, {headers:{"Content-Type":"application/json"}, ...options});
  if(!res.ok) throw new Error(await res.text());
  if(res.status===204) return null;
  return res.json();
}

function App(){
  const [tab,setTab]=useState("overview");
  const [data,setData]=useState({pages:[],posts:[],tags:[],menus:[],dashboard:{}});
  const [error,setError]=useState("");
  async function load(){
    setError("");
    try{
      const [dashboard,pages,posts,tags,menus]=await Promise.all([api("/api/dashboard"),api("/api/pages"),api("/api/posts"),api("/api/tags"),api("/api/menus")]);
      setData({dashboard,pages,posts,tags,menus});
    }catch(err){setError(String(err.message||err));}
  }
  useEffect(()=>{load();},[]);
  return <div className="shell">
    <aside className="side">
      <div className="brand">GION CMS</div>
      <div className="nav">{["overview","pages","posts","tags","menus"].map(name=><button key={name} className={tab===name?"active":""} onClick={()=>setTab(name)}>{name[0].toUpperCase()+name.slice(1)}</button>)}</div>
    </aside>
    <main className="main">
      <section className="hero"><div><h1>Editorial control room</h1><p>Create pages, publish posts, assign tags, and wire navigation entries for the public GION/v2 site.</p></div><a className="button" href="/" target="_blank">View site</a></section>
      {error && <div className="panel" style={{borderColor:"#ef4444"}}>{error}</div>}
      {tab==="overview" && <Overview dashboard={data.dashboard}/>} 
      {tab==="pages" && <Resource title="Pages" rows={data.pages} empty={emptyPage} endpoint="/api/pages" onLoad={load} renderForm={PageForm}/>} 
      {tab==="posts" && <Resource title="Posts" rows={data.posts} empty={emptyPost} endpoint="/api/posts" onLoad={load} renderForm={(props)=><PostForm {...props} tags={data.tags}/>}/>} 
      {tab==="tags" && <Resource title="Tags" rows={data.tags} empty={emptyTag} endpoint="/api/tags" onLoad={load} renderForm={TagForm}/>} 
      {tab==="menus" && <Resource title="Menus" rows={data.menus} empty={emptyMenu} endpoint="/api/menus" onLoad={load} renderForm={(props)=><MenuForm {...props} pages={data.pages} tags={data.tags}/>}/>} 
    </main>
  </div>;
}

function Overview({dashboard}){return <div className="cards">{Object.entries(dashboard||{}).map(([k,v])=><div className="stat" key={k}><strong>{v}</strong><span>{k}</span></div>)}</div>}

function Resource({title, rows, empty, endpoint, onLoad, renderForm}){
  const [editing,setEditing]=useState(null);
  const current = editing || empty;
  async function save(value){await api(value.ID?`${endpoint}/${value.ID}`:endpoint,{method:value.ID?"PUT":"POST",body:JSON.stringify(value)});setEditing(null);await onLoad();}
  async function remove(row){if(confirm(`Delete ${row.title||row.name||row.label}?`)){await api(`${endpoint}/${row.ID}`,{method:"DELETE"});await onLoad();}}
  return <div className="grid"><section className="panel"><div className="toolbar"><h2>{title}</h2><button className="button" onClick={()=>setEditing({...empty})}>New</button></div><div className="list">{rows.map(row=><div className="item" key={row.ID}><div><h3>{row.title||row.name||row.label}</h3><p>{row.slug||row.kind||"content"} {row.published!==undefined && <span className="pill">{row.published?"published":"draft"}</span>}</p></div><div className="row"><button className="button ghost" onClick={()=>setEditing({...row})}>Edit</button><button className="button danger" onClick={()=>remove(row)}>Delete</button></div></div>)}</div></section><section className="panel"><h2>{current.ID?"Edit":"Create"} {title.slice(0,-1)}</h2>{renderForm({value:current,onChange:setEditing,onSave:save})}</section></div>
}

function Field({label, children}){return <label className="field"><span>{label}</span>{children}</label>}
function Text({label,value,onChange,name,area=false}){const Tag=area?"textarea":"input";return <Field label={label}><Tag value={value[name]||""} onChange={e=>onChange({...value,[name]:e.target.value})}/></Field>}
function Published({value,onChange}){return <Field label="Published"><select value={value.published?"yes":"no"} onChange={e=>onChange({...value,published:e.target.value==="yes"})}><option value="yes">Yes</option><option value="no">No</option></select></Field>}

function PageForm({value,onChange,onSave}){return <form onSubmit={e=>{e.preventDefault();onSave(value)}}><Text label="Title" name="title" value={value} onChange={v=>onChange({...v,slug:v.slug||slugify(v.title)})}/><Text label="Slug" name="slug" value={value} onChange={onChange}/><Text label="Summary" name="summary" value={value} onChange={onChange}/><Text label="Cover image URL" name="coverImage" value={value} onChange={onChange}/><Text label="Gallery image URLs" name="images" value={value} onChange={onChange} area/><Text label="HTML body" name="body" value={value} onChange={onChange} area/><Published value={value} onChange={onChange}/><button className="button">Save page</button></form>}
function TagForm({value,onChange,onSave}){return <form onSubmit={e=>{e.preventDefault();onSave(value)}}><Text label="Name" name="name" value={value} onChange={v=>onChange({...v,slug:v.slug||slugify(v.name)})}/><Text label="Slug" name="slug" value={value} onChange={onChange}/><button className="button">Save tag</button></form>}
function PostForm({value,onChange,onSave,tags}){function toggle(tag){const exists=(value.tags||[]).some(t=>t.ID===tag.ID);onChange({...value,tags:exists?(value.tags||[]).filter(t=>t.ID!==tag.ID):[...(value.tags||[]),tag]});}return <form onSubmit={e=>{e.preventDefault();onSave(value)}}><Text label="Title" name="title" value={value} onChange={v=>onChange({...v,slug:v.slug||slugify(v.title)})}/><Text label="Slug" name="slug" value={value} onChange={onChange}/><Text label="Summary" name="summary" value={value} onChange={onChange}/><Text label="Cover image URL" name="coverImage" value={value} onChange={onChange}/><Text label="Gallery image URLs" name="images" value={value} onChange={onChange} area/><Text label="HTML body" name="body" value={value} onChange={onChange} area/><Text label="Right column HTML" name="rightBody" value={value} onChange={onChange} area/><Field label="Tags"><div className="row">{tags.map(tag=><button type="button" key={tag.ID} className="button ghost" style={{outline:(value.tags||[]).some(t=>t.ID===tag.ID)?"2px solid var(--ok)":"none"}} onClick={()=>toggle(tag)}>{tag.name}</button>)}</div></Field><Published value={value} onChange={onChange}/><button className="button">Save post</button></form>}
function MenuForm({value,onChange,onSave,pages,tags}){return <form onSubmit={e=>{e.preventDefault();onSave(value)}}><Text label="Label" name="label" value={value} onChange={onChange}/><Field label="Kind"><select value={value.kind} onChange={e=>onChange({...value,kind:e.target.value})}><option value="page">Page</option><option value="tag">Post tag list</option><option value="url">Custom URL</option></select></Field>{value.kind==="page"&&<Field label="Page"><select value={value.pageId||""} onChange={e=>onChange({...value,pageId:Number(e.target.value)||null})}><option value="">Select page</option>{pages.map(p=><option key={p.ID} value={p.ID}>{p.title}</option>)}</select></Field>}{value.kind==="tag"&&<Field label="Tag"><select value={value.tagId||""} onChange={e=>onChange({...value,tagId:Number(e.target.value)||null})}><option value="">Select tag</option>{tags.map(t=><option key={t.ID} value={t.ID}>{t.name}</option>)}</select></Field>}{value.kind==="url"&&<Text label="URL" name="url" value={value} onChange={onChange}/>}<Field label="Position"><input type="number" value={value.position||0} onChange={e=>onChange({...value,position:Number(e.target.value)})}/></Field><Field label="Visible"><select value={value.visible?"yes":"no"} onChange={e=>onChange({...value,visible:e.target.value==="yes"})}><option value="yes">Yes</option><option value="no">No</option></select></Field><button className="button">Save menu item</button></form>}

ReactDOM.createRoot(document.getElementById("root")).render(<App/>);
