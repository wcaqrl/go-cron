(window.webpackJsonp=window.webpackJsonp||[]).push([[7],{"659C":function(t,e,i){"use strict";i.r(e),i.d(e,"LogModule",function(){return P});var s=i("M0ag"),a=i("fXoL"),r=i("Ac7g"),n=i("PScX"),o=i("dEAy"),p=i("SdXu"),c=i("JA5x"),l=i("ok2U"),d=i("DGaY");const m=["st"];function h(t,e){}const g=function(){return{x:"1800px",y:"480px"}};let u=(()=>{class t{constructor(t,e,i,s,a){this.http=t,this.modal=e,this.msg=i,this.modalSrv=s,this.cdr=a,this.url="/api/log/list",this.totalUrl="/api/log/count",this.originPerpage=15,this.total=0,this.q={project:"orange",page:1,perpage:this.originPerpage,ip_addr:"",domain:"",http_method:"",http_code:0,response_time:0,request_route:"",request_param:"",user_agent:"",start_time:0,end_time:0,sort:""},this.pagination={front:!1,position:"bottom",placement:"right",show:!0,total:!0},this.loading=!1,this.searchSchema={properties:{project:{type:"string",title:"\u9879\u76ee\u540d\u79f0",enum:[{label:"orange",value:"orange"},{label:"open-api",value:"open-api"},{label:"tvapi",value:"tvapi"},{label:"tvcms",value:"orange"}],default:"orange",ui:{widget:"select"}},ip_addr:{type:"string",title:"IP\u5730\u5740"},domain:{type:"string",title:"\u57df\u540d"},http_method:{type:"string",title:"\u8bf7\u6c42\u65b9\u6cd5"},http_code:{type:"number",title:"\u54cd\u5e94\u7801"},response_time:{type:"number",title:"\u54cd\u5e94\u65f6\u957f"},request_route:{type:"string",title:"\u8bf7\u6c42\u8def\u7531"},request_param:{type:"string",title:"\u8bf7\u6c42\u53c2\u6570"},user_agent:{type:"string",title:"\u8bf7\u6c42\u4ee3\u7406"},start_time:{type:"string",title:"\u5f00\u59cb\u65f6\u95f4",ui:{widget:"date",mode:"date",showTime:!0,displayFormat:"yyyy-MM-dd HH:mm:ss",format:"t"}},end_time:{type:"string",title:"\u622a\u6b62\u65f6\u95f4",ui:{widget:"date",mode:"date",showTime:!0,displayFormat:"yyyy-MM-dd HH:mm:ss",format:"t"}}}},this.columns=[{title:"ID",index:"id"},{title:"\u771f\u5b9eIP",index:"real_ip"},{title:"\u8bf7\u6c42\u65f6\u95f4",type:"date",dateFormat:"yyyy-MM-dd HH:mm:ss",index:"request_time"},{title:"\u8bf7\u6c42\u57df\u540d",index:"domain"},{title:"\u8bf7\u6c42\u65b9\u6cd5",index:"http_method"},{title:"\u8bf7\u6c42\u8def\u7531",index:"request_route"},{title:"\u8bf7\u6c42\u53c2\u6570",index:"request_param"},{title:"\u8bf7\u6c42\u4ee3\u7406",index:"user_agent"},{title:"\u54cd\u5e94\u4ee3\u7801",index:"http_code"},{title:"\u54cd\u5e94\u65f6\u95f4",index:"response_time"}]}ngOnInit(){this.getData()}getData(){this.getList(),this.getTotal()}getList(){this.http.get(this.url,this.filterParams(this.q)).subscribe(t=>{this.data=t,this.cdr.detectChanges()})}getTotal(){this.http.get(this.totalUrl,this.filterParams(this.q)).subscribe(t=>{this.total=t.hasOwnProperty("count")?t.count:0,this.cdr.detectChanges()})}filterParams(t){const e={};for(const i in t)t.hasOwnProperty(i)&&t[i]&&(e[i]=t[i]);return e}submit(t){setTimeout(()=>{for(const e in t)t.hasOwnProperty(e)&&this.q.hasOwnProperty(e)&&(this.q[e]=t[e]);this.getData()})}reset(){setTimeout(()=>{this.q={project:"orange",page:1,perpage:this.originPerpage,ip_addr:"",domain:"",http_method:"",http_code:0,response_time:0,request_route:"",request_param:"",user_agent:"",start_time:0,end_time:0,sort:""},this.getData()})}change(t){"click"!==t.type&&"dblClick"!==t.type&&"loaded"!==t.type&&("pi"===t.type||"ps"===t.type?(this.q.page=t.pi,this.q.perpage=t.ps,this.getList()):(this.q.page=t.pi,this.q.perpage=t.ps,this.getData()))}}return t.\u0275fac=function(e){return new(e||t)(a.Rb(r.p),a.Rb(r.i),a.Rb(n.e),a.Rb(o.c),a.Rb(a.h))},t.\u0275cmp=a.Lb({type:t,selectors:[["app-log-list"]],viewQuery:function(t,e){if(1&t&&a.Pc(m,1),2&t){let t;a.xc(t=a.fc())&&(e.st=t.first)}},decls:7,vars:11,consts:[[3,"action"],["phActionTpl",""],["mode","search",3,"schema","formSubmit","formReset"],["virtualScroll","",3,"scroll","data","columns","loading","pi","ps","total","page","change"],["st",""]],template:function(t,e){if(1&t){const t=a.Yb();a.Xb(0,"page-header",0),a.Jc(1,h,0,0,"ng-template",null,1,a.Kc),a.Wb(),a.Xb(3,"nz-card"),a.Xb(4,"sf",2),a.ec("formSubmit",function(t){return e.submit(t)})("formReset",function(e){return a.Bc(t),a.yc(6).reset(e)}),a.Wb(),a.Xb(5,"st",3,4),a.ec("change",function(t){return e.change(t)}),a.Wb(),a.Wb()}if(2&t){const t=a.yc(2);a.qc("action",t),a.Db(4),a.qc("schema",e.searchSchema),a.Db(1),a.qc("scroll",a.sc(10,g))("data",e.data)("columns",e.columns)("loading",e.loading)("pi",e.q.page)("ps",e.q.perpage)("total",e.total)("page",e.pagination)}},directives:[p.a,c.a,l.b,d.a],encapsulation:2}),t})();const y=["st"];function b(t,e){}let f=(()=>{class t{constructor(t,e,i,s,a){this.http=t,this.modal=e,this.msg=i,this.modalSrv=s,this.cdr=a,this.url="/job/log/list",this.totalUrl="/job/log/count",this.originPerpage=15,this.total=0,this.q={page:1,perpage:this.originPerpage,name:"",ip_addr:"",command:"",errorCode:0,msg:"",traceID:"",start_time:0,end_time:0,sort:""},this.pagination={front:!1,position:"bottom",placement:"right",show:!0,total:!0},this.loading=!1,this.searchSchema={properties:{name:{type:"string",title:"\u4efb\u52a1\u540d\u79f0"},ip_addr:{type:"string",title:"ip\u5730\u5740"},command:{type:"string",title:"Shell\u547d\u4ee4"},errorCode:{type:"number",title:"\u9519\u8bef\u7801"},msg:{type:"string",title:"\u9519\u8bef\u6d88\u606f"},traceID:{type:"string",title:"\u8ffd\u8e2aID"},start_time:{type:"string",title:"\u5f00\u59cb\u65f6\u95f4",ui:{widget:"date",mode:"date",showTime:!0,displayFormat:"yyyy-MM-dd HH:mm:ss",format:"t"}},end_time:{type:"string",title:"\u622a\u6b62\u65f6\u95f4",ui:{widget:"date",mode:"date",showTime:!0,displayFormat:"yyyy-MM-dd HH:mm:ss",format:"t"}}}},this.columns=[{title:"IP\u5730\u5740",index:"ip_addr"},{title:"\u4efb\u52a1\u540d\u79f0",index:"job_name"},{title:"Shell\u547d\u4ee4",index:"command"},{title:"\u8c03\u7528\u7ed3\u679c",index:"err"},{title:"\u8ba1\u5212\u65f6\u95f4",type:"date",dateFormat:"yyyy-MM-dd HH:mm:ss.SSS",index:"plan_time"},{title:"\u8c03\u5ea6\u65f6\u95f4",type:"date",dateFormat:"yyyy-MM-dd HH:mm:ss.SSS",index:"schedule_time"},{title:"\u5f00\u59cb\u65f6\u95f4",type:"date",dateFormat:"yyyy-MM-dd HH:mm:ss.SSS",index:"start_time"},{title:"\u7ed3\u675f\u65f6\u95f4",type:"date",dateFormat:"yyyy-MM-dd HH:mm:ss.SSS",index:"end_time"},{title:"\u8ffd\u8e2aID",index:"traceID"},{title:"\u9519\u8bef\u7801",index:"errorCode"},{title:"\u9519\u8bef\u6d88\u606f",index:"msg"}]}ngOnInit(){this.getData()}getData(){this.getList(),this.getTotal()}getList(){this.http.get(this.url,this.filterParams(this.q)).subscribe(t=>{this.data=t,this.cdr.detectChanges()})}getTotal(){this.http.get(this.totalUrl,this.filterParams(this.q)).subscribe(t=>{this.total=t.hasOwnProperty("count")?t.count:0,this.cdr.detectChanges()})}filterParams(t){const e={};for(const i in t)t.hasOwnProperty(i)&&t[i]&&(e[i]=t[i]);return e}submit(t){setTimeout(()=>{for(const e in t)t.hasOwnProperty(e)&&this.q.hasOwnProperty(e)&&(this.q[e]=t[e]);this.getData()})}reset(){setTimeout(()=>{this.q={page:1,perpage:this.originPerpage,name:"",ip_addr:"",command:"",errorCode:0,msg:"",traceID:"",start_time:0,end_time:0,sort:""},this.getData()})}change(t){"click"!==t.type&&"dblClick"!==t.type&&"loaded"!==t.type&&("pi"===t.type||"ps"===t.type?(this.q.page=t.pi,this.q.perpage=t.ps,this.getList()):(this.q.page=t.pi,this.q.perpage=t.ps,this.getData()))}}return t.\u0275fac=function(e){return new(e||t)(a.Rb(r.p),a.Rb(r.i),a.Rb(n.e),a.Rb(o.c),a.Rb(a.h))},t.\u0275cmp=a.Lb({type:t,selectors:[["app-log-list"]],viewQuery:function(t,e){if(1&t&&a.Pc(y,1),2&t){let t;a.xc(t=a.fc())&&(e.st=t.first)}},decls:7,vars:9,consts:[[3,"action"],["phActionTpl",""],["mode","search",3,"schema","formSubmit","formReset"],[3,"data","columns","loading","pi","ps","total","page","change"],["st",""]],template:function(t,e){if(1&t&&(a.Xb(0,"page-header",0),a.Jc(1,b,0,0,"ng-template",null,1,a.Kc),a.Wb(),a.Xb(3,"nz-card"),a.Xb(4,"sf",2),a.ec("formSubmit",function(t){return e.submit(t)})("formReset",function(){return e.reset()}),a.Wb(),a.Xb(5,"st",3,4),a.ec("change",function(t){return e.change(t)}),a.Wb(),a.Wb()),2&t){const t=a.yc(2);a.qc("action",t),a.Db(4),a.qc("schema",e.searchSchema),a.Db(1),a.qc("data",e.data)("columns",e.columns)("loading",e.loading)("pi",e.q.page)("ps",e.q.perpage)("total",e.total)("page",e.pagination)}},directives:[p.a,c.a,l.b,d.a],encapsulation:2}),t})();var _=i("tyNb");const q=[{path:"job",component:f},{path:"http",component:u}];let w=(()=>{class t{}return t.\u0275fac=function(e){return new(e||t)},t.\u0275mod=a.Pb({type:t}),t.\u0275inj=a.Ob({imports:[[_.m.forChild(q)],_.m]}),t})(),P=(()=>{class t{}return t.\u0275fac=function(e){return new(e||t)},t.\u0275mod=a.Pb({type:t}),t.\u0275inj=a.Ob({imports:[[s.b,w]]}),t})()}}]);