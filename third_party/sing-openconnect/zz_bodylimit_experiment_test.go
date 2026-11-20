package openconnect
import("reflect"; "runtime"; "strings"; "testing"; "time")
func TestBodyLimitExperiment(t *testing.T){
 for _, n := range []int{(2<<20)+1,8<<20,(16<<20)-1024} {
  doc:=[]byte(`<sslvpn-tunnel><ipv4><assigned-addr ipv4='192.0.2.8'/></ipv4><ignored note='`+strings.Repeat("x",n)+`'/></sslvpn-tunnel>`)
  base,e:=parseFortinetBaseline(doc,time.Unix(0,0)); if e!=nil {t.Fatal(e)}
  runtime.GC(); var a,b runtime.MemStats; runtime.ReadMemStats(&a)
  got,e:=parseFortinetXMLConfiguration(doc,time.Unix(0,0)); runtime.ReadMemStats(&b)
  if e!=nil || !reflect.DeepEqual(base,got) {t.Fatalf("n=%d err=%v",n,e)}
  t.Logf("input=%d cumulative_alloc=%d heap_alloc_after=%d matches_baseline=true",len(doc),b.TotalAlloc-a.TotalAlloc,b.HeapAlloc)
 }
}
func TestBodyLimitLexicalFormsBaseline(t *testing.T) {
 prefix:=`<sslvpn-tunnel><ipv4><assigned-addr ipv4='192.0.2.8'/></ipv4>`; suffix:=`</sslvpn-tunnel>`; x:=strings.Repeat("z",(2<<20)+16)
 forms:=map[string]string{"attr":`<ignored a='`+x+`'/>`,"text":`<ignored>`+x+`</ignored>`,"comment":`<!--`+x+`-->`,"cdata":`<ignored><![CDATA[`+x+`]]></ignored>`,"pi":`<?example `+x+`?>`}
 for name, form:=range forms { t.Run(name,func(t *testing.T){doc:=[]byte(prefix+form+suffix); now:=time.Unix(0,0); base,e:=parseFortinetBaseline(doc,now); got,e2:=parseFortinetXMLConfiguration(doc,now); if e!=nil||e2!=nil||!reflect.DeepEqual(base,got){t.Fatalf("baseline=%v new=%v",e,e2)}; t.Logf("bytes=%d matches_baseline",len(doc))}) }
 doc:=make([]byte,fortinetMaximumAuthenticationBody+1); if preflightFortinetXMLBudget(doc)==nil {t.Fatal("body limit not enforced")}
}
