# 鑻遍泟寮曟搸瀹炵幇璁″垝 / Hero Engine Implementation Plan

> 鐘舵€侊細2026-08-21銆傚紩鎿庡凡瀹炵幇 21 涓嫳闆勯潰锛屾湭瀹炵幇 51 涓紙绾?46 涓嫭绔嬭嫳闆勶級銆?> 鏈枃妗ｅ崗璋冮獞澹洟鍐呯殑瀹炵幇鍒嗗伐锛屽姩鎵嬪墠鍏堣棰嗐€佸姩鎵嬪悗鎵撳嬀锛岄伩鍏嶆挒杞︺€?
## 宸插疄鐜帮紙21锛?
鏍稿績鐩掞細铚樿洓渚犮€佹儕濂囬槦闀裤€佸コ缁垮法浜恒€侀挗閾佷緺銆侀粦璞癸紱缇庡浗闃熼暱銆佹儕濂囧皯濂炽€佸寮傚崥澹€佹牸椴佺壒銆佺伀绠担鐔娿€侀挗鍔涘＋銆佸够褰辩尗銆佺數绱€佸绫宠銆佸ぉ浣?澶уぉ浣裤€佸琛岃€呫€佽檸濂炽€佹旦鍏嬪皬瀛愩€佸榄斾緺銆佸洖澹般€?
## 鏈疄鐜版竻鍗曪紙51 涓嫳闆勯潰锛?
| 娉㈡ | 鍗″彿 | 鑻遍泟 | 鍖?| 鏈哄埗瑕佺偣 | 鐘舵€?|
|---|---|---|---|---|---|
| W1 | 06001a | 绱㈠皵 Thor | thor | 濡欏皵灏煎皵妫€绱紝绠€鍗曟暟鍊?| 鉁?kaguya+shantu |
| W1 | 10001a | 娴╁厠 Hulk | hlk | 鍙樿韩澧炰激锛岀畝鍗曟暟鍊?| 馃敤 kaguya |
| W1 | 04001a | 楣扮溂 Hawkeye | trors | 寮?绠煝浣撶郴锛堢鐭?attack/thwart 浜嬩欢锛?| 馃敤 shantu |
| W1 | 04031a | 铚樿洓濂充緺 Spider-Woman | trors | 鍙屾淳绯绘瀯绛戯紙鏋勭瓚鏈熻鍒欙紝寮曟搸鍐呭彲绠€鍖栵級 | 猬?|
| W2 | 08001a | 榛戝濡?Black Widow | bkw | Preparation 鍑嗗寮傝兘浣撶郴 | 猬?|
| W2 | 14001a | 蹇摱 Quicksilver | qsv | 鎶界墝/寮冪墝寰幆 | 猬?|
| W2 | 15001a | 缁孩濂冲帆 Scarlet Witch | scw | 娣锋矊鎺у埗 | 猬?|
| W2 | 23001a | 鎴樹簤鏈哄櫒 War Machine | warm | 寮硅嵂鏍囪绠＄悊 | 猬?|
| W2 | 28001a | 鏂版槦 Nova | nova | 瓒呮柊鏄熷ご鐩斿舰鎬佸垏鎹?| 猬?|
| W3 | 50001a | 鐜涗附浜毬峰笇灏?Maria Hill | aos | 鎴樿。褰㈡€佸崌绾э紙suit form锛変綋绯?| 猬?|
| W3 | 50034a | 灏煎厠路寮楃憺 Nick Fury | aos | 鎴樿。褰㈡€侊紙绐佸嚮/娼滆鍒囨崲锛?| 猬?|
| W3 | 12001a/c | 铓佷汉 Ant-Man | ant | 澶氬舰鎬侊紙宸ㄤ汉/宸ㄥ寲锛?| 猬?|
| W3 | 13001a/c | 榛勮渹濂?Wasp | wsp | 澶氬舰鎬?| 猬?|
| W4 | 17001a | 鏄熺埖 Star-Lord | stld | 鍏冪礌鏋?| 猬?|
| W4 | 18001a | 鍗￠瓟鎷?Gamora | gam | 鏀诲嚮/鍖栬В浜嬩欢璁℃暟 | 猬?|
| W4 | 19001a | 寰锋媺鍏嬫柉 Drax | drax | 澶嶄粐鏍囪 | 猬?|
| W4 | 20001a | 姣掓恫 Venom | vnm | 姝﹀櫒闄愬埗鏁?1 | 猬?|
| W4 | 21001a | 鍏夎氨 Spectrum | mts | 鑳介噺褰㈡€佷笁鍗＄炕杞?| 猬?|
| W4 | 21031a | 浜氬綋鏈＋ Adam Warlock | mts | 鏋勭瓚闄愬埗锛堝洓娲剧郴鍧囩瓑锛?| 猬?|
| W4 | 22001a | 鏄熶簯 Nebula | nebu | 鎶€宸у崌绾х粨绠椾綋绯?| 猬?|
| W5 | 25001a | 濂虫绁?Valkyrie | valk | 姝讳骸涔嬪厜 | 猬?|
| W5 | 26001a | 骞昏 Vision | vision | 璐ㄩ噺褰㈡€侊紙鑷村瘑/鏃犲舰锛?| 猬?|
| W5 | 27001a | 骞界伒铚樿洓 Ghost-Spider | sm | 澶氬厓瀹囧畽闂ㄧエ | 猬?|
| W5 | 27030a | 铚樿洓渚?Spider-Man | sm | 铔涚綉鍙戝皠鍣ㄤ綋绯?| 猬?|
| W5 | 29001a-29003a | 閽㈤搧涔嬪績 Ironheart | ironheart | 涓夎韩浠?杩涘害鏍囪鍗囩骇 | 猬?|
| W5 | 30001a | 铚樿洓渚犳眽濮?Spider-Ham | spiderham | 鍗￠€氭爣璁?| 猬?|
| W5 | 31001a | SP//dr鎴樼敳 | spdr | 浣╁Ξ鍒嗙/鍚堜綋 | 猬?|
| W6 | 33001a | 闀皠鐪?Cyclops | cyclops | 鎴樻湳鍗囩骇+璺ㄦ淳绯荤洘鍙?| 猬?|
| W6 | 34001a | 鍑ゅ嚢 Phoenix | phoenix | 鍑ゅ嚢涔嬪姏鍔涢噺鏍囪 | 猬?|
| W6 | 35001a | 閲戝垰鐙?Wolverine | wolv | 鐖?鑷剤 | 猬?|
| W6 | 36001a | 鏆撮濂?Storm | storm | 澶╂皵鐗屽爢 | 猬?|
| W6 | 37001a | 鐗岀殗 Gambit | gambit | 鐩楄醇妫€瑙?| 猬?|
| W6 | 38001a | 缃楀埞濂?Rogue | rogue | 瑙︾鍗囩骇 | 猬?|
| W7 | 41001a | 鐏佃澏 Psylocke | psylocke | 鐏佃兘鍙屽崌绾х炕杞?| 猬?|
| W7 | 43001a | X-23 | x23 | 鐖?娲楀洖寰幆 | 猬?|
| W7 | 44001a | 姝讳緧 Deadpool | deadpool | 绗洓闈㈠妫€绱?| 猬?|
| W7 | 45001a | 姣曡倴鏅?Bishop | aoa | 鏃堕棿鐗屽爢 | 猬?|
| W7 | 45030a | 绉樺 Magik | aoa | 娉曟湳鐗屽爢 | 猬?|
| W8 | 46001a | 鍐颁汉 Iceman | iceman | 鍐讳激鍗囩骇 | 猬?|
| W8 | 47001a | 鍗冩 Jubilee | jubilee | 璐墿瀵嗚皨 | 猬?|
| W8 | 49001a | 涓囩鐜?Magneto | magneto | 纾佸姏浣撶郴 | 猬?|
| W8 | 51001a | 榛戣惫 Black Panther | bp | 鍙戞槑瀹舵绱?| 猬?|
| W8 | 52001a | 涓?Silk | silk | 濉炲崱鏈哄埗 | 猬?|
| W8 | 53001a | 鐚庨拱 Falcon | falcon | 楦熷崱浣撶郴 | 猬?|
| W8 | 54001a | 鍐棩鎴樺＋ Winter Soldier | winter | 鏈烘鑷?| 猬?|
| W8 | 58001a | 濂囪抗浜?Wonder Man | wonder_man | 绂诲瓙鐢熺悊 | 猬?|
| W8 | 59001a | 璧媺鍏嬪嫆鏂?Hercules | hercules | 璇曠偧/绀肩墿鍙岀墝鍫?| 猬?|

## 瀹炵幇瑙勮寖锛堟瘡涓嫳闆勶級

1. **鏂板寘**锛歚internal/engine/cards/<pack>/`锛堝凡瀛樺湪鐨勫寘鐩存帴鍔犳枃浠讹級
   - `<hero>.go`锛氳韩浠借涓?`engine.RegisterBehavior("<base>", &engine.Behavior{...})`
   - `signatures.go`锛氫笓灞炲崱琛屼负
   - 瀹挎晫/閲嶆媴鎸夌幇鏈夊寘鐨勬ā寮忥紙鍙傝€?`msmarvel/`锛?2. **Behavior 閽╁瓙**锛堣 `internal/engine/entity.go`锛夛細
   - `HeroAbilities` / `AlterEgoAbilities`锛氫富鍔ㄥ紓鑳?   - `React`锛氬搷搴?鎵撴柇锛堢洃鍚?engine.Message锛?   - `CardCost` / `IdentityStats` / `Resource` 绛夎鍔ㄩ挬瀛?   - 鐜╁鎶夋嫨涓€寰嬬敤 `engine.AskQuestion` + `engine.Choice`
3. **娉ㄥ唽**锛歚init()` 閲屾敞鍐岋紱`cmd/server/main.go` 鍔?blank import
4. **娴嬭瘯**锛歚<pack>_test.go` 瑕嗙洊鏍稿績寮傝兘瑙﹀彂锛堝弬鑰?`msmarvel/msm_test.go`锛?5. **楠岃瘉**锛歚go build ./... && go test ./internal/engine/...`锛屾湰鍦拌捣鏈嶅姟纭 `/api/v1/marvel/cards` 鏃犲洖褰?6. **鍗＄墝鏂囨湰**锛氱炕璇戝眰宸茶鐩栵紙tools/zh/out锛夛紝琛屼负浠ｇ爜閲屽紩鐢ㄥ崱鍚嶇敤鑻辨枃鍘熸枃锛堝紩鎿庡唴閮級锛屾樉绀哄眰鑷姩涓枃鍖?
## 鍒嗗伐绾﹀畾

- 鍔ㄦ墜鍓嶆妸鐘舵€佸垪 猬?鏀规垚 馃敤锛堜綘鐨勫悕瀛楋級锛屽畬鎴愬悗鏀?鉁?- 姣忎釜鑻遍泟涓€涓嫭绔嬫彁浜わ紝鎻愪氦淇℃伅 `feat(hero): implement <name> (<code>)`
- 鎺ㄩ€佸墠 `git fetch origin && git rebase origin/main`锛坉ynilath 娲昏穬寮€鍙戜腑锛?- W1-W2 鐢?shantu + kaguya 骞惰鎺ㄨ繘锛沇3 aos 鍙岃嫳闆勪紭鍏堬紙鐜╁鐐瑰悕瑕佺帺锛?
