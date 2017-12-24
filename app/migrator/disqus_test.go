package migrator

import (
	"strings"
	"testing"

	"time"

	"github.com/stretchr/testify/assert"
	"github.com/umputun/remark/app/store"
)

func TestDisqus_Convert(t *testing.T) {
	d := Disqus{}
	ch := d.convert(strings.NewReader(xmlTest), "test")

	res := []store.Comment{}
	for comment := range ch {
		res = append(res, comment)
		t.Logf("%+v", comment)
	}
	assert.Equal(t, 3, len(res), "3 comments total")

	exp0 := store.Comment{
		ID: "3565798471341011339",
		Locator: store.Locator{
			SiteID: "test",
			URL:    "https://radio-t.com/p/2011/03/05/podcast-229/",
		},
		Text: "<p>The quick brown fox jumps over the lazy dog.</p>",
		User: store.User{
			Name: "Alexander Puzatykh",
			ID:   "facebook-1787732238",
			IP:   "178.234.205.125",
		},
	}
	exp0.Timestamp, _ = time.Parse("2006-01-02T15:04:05Z", "2011-08-31T15:16:29Z")
	assert.Equal(t, exp0, res[0])
}

var xmlTest = `
<?xml version="1.0" encoding="utf-8"?>
<disqus xmlns="http://disqus.com" xmlns:dsq="http://disqus.com/disqus-internals" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://disqus.com/api/schemas/1.0/disqus.xsd http://disqus.com/api/schemas/1.0/disqus-internals.xsd">

	<thread dsq:id="247918464">
		<id/>
		<forum>radiot</forum>
		<category dsq:id="707279"/>
		<link>http://radio-t.umputun.com/2011/03/229_8880.html</link>
		<title>Радио-Т: Радио-Т 229</title>
		<message/>
		<createdAt>2011-03-07T20:46:25Z</createdAt>
		<author>
			<email>umputun@gmail.com</email>
			<name>Umputun</name>
			<isAnonymous>false</isAnonymous>
			<username>umputun</username>
		</author>
		<ipAddress>98.212.28.115</ipAddress>
		<isClosed>false</isClosed>
		<isDeleted>false</isDeleted>
	</thread>
	<thread dsq:id="247937687">
		<id>http://www.radio-t.com/p/2011/03/05/podcast-229/</id>
		<forum>radiot</forum>
		<category dsq:id="707279"/>
		<link>https://radio-t.com/p/2011/03/05/podcast-229/</link>
		<title>Радио-Т: Радио-Т 229</title>
		<message/>
		<createdAt>2011-03-07T21:17:17Z</createdAt>
		<author>
			<email>umputun@gmail.com</email>
			<name>Umputun</name>
			<isAnonymous>false</isAnonymous>
			<username>umputun</username>
		</author>
		<ipAddress>80.250.214.235</ipAddress>
		<isClosed>true</isClosed>
		<isDeleted>false</isDeleted>
	</thread>

	<post dsq:id="299619020">
		<id>3565798471341011339</id>
		<message>
			<![CDATA[<p>The quick brown fox jumps over the lazy dog.</p>]]>
		</message>
		<createdAt>2011-08-31T15:16:29Z</createdAt>
		<isDeleted>false</isDeleted>
		<isSpam>false</isSpam>
		<author>
			<email/>
			<name>Alexander Puzatykh</name>
			<isAnonymous>false</isAnonymous>
			<username>facebook-1787732238</username>
		</author>
		<ipAddress>178.234.205.125</ipAddress>
		<thread dsq:id="247937687"/>
	</post>

	<post dsq:id="299744309">
		<id>3029154520436241933</id>
		<message>
			<![CDATA[<p>Microsoft показал проводник Windows 8 с ленточным интерфейсом.</p><p><a href="http://blogs.msdn.com/b/b8/archive/2011/08/29/improvements-in-windows-explorer.aspx" rel="nofollow noopener" title="http://blogs.msdn.com/b/b8/archive/2011/08/29/improvements-in-windows-explorer.aspx">http://blogs.msdn.com/b/b8/...</a> </p>]]>
		</message>
		<createdAt>2011-08-31T17:44:22Z</createdAt>
		<isDeleted>false</isDeleted>
		<isSpam>false</isSpam>
		<author>
			<email>mihail.merkulov@gmail.com</email>
			<name>mikhailmerkulov</name>
			<isAnonymous>false</isAnonymous>
			<username>mikhailmerkulov</username>
		</author>
		<ipAddress>195.234.75.139</ipAddress>
		<thread dsq:id="247937687"/>
	</post>

	<post dsq:id="299986072">
		<id>6580890074280459209</id>
		<message>
			<![CDATA[<p>Google App Engine скоро выходит из превью статуса.</p><p>Сейчас письмо пришло от гугла.</p><p>Для платных приложений использущих High Replication Datastore (HRD) будет 99.95% uptime SLA.<br>Будут Премьер аккаунты за 500 баксов/месяц с оперативной поддержкой и любым количеством приложений на аккаунте (+ плата за потребленные ресурсы).<br>В связи с переходом на новую систему оплаты, обещают снизить бесплатные квоты.<br>Всем кто включит биллинг до 31 октября, обещают 50 баксов :)</p>]]>
		</message>
		<createdAt>2011-08-31T22:48:43Z</createdAt>
		<isDeleted>false</isDeleted>
		<isSpam>false</isSpam>
		<author>
			<email>unikier@gmail.com</email>
			<name>Dmitry Shapoval</name>
			<isAnonymous>false</isAnonymous>
			<username>google-74b9e7568ef6860e93862c5d7752b657</username>
		</author>
		<ipAddress>89.113.25.139</ipAddress>
		<thread dsq:id="247937687"/>
	</post>
</disqus>
`
