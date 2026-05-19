# TODO
-- wiki-rag中目前仍有一些query是存在歧义的，例如”东山“在原文中指代的是京都东山，但返回的释义是今山东蒙山；
    也有一些错误的，例如将“北海道”切割成“北海去query。
-- 制定一个comparison标准，来反馈出rag的输出是否是enrich了文本，给出了更加丰富、准确的释义和科普。
# DONE
-- 制定rag/non-rag的测评体系，metrics包括query, query document, rag context, latency。
