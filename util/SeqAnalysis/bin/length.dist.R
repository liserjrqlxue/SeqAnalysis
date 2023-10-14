#! /usr/bin/Rscript
library(ggplot2)
library(stringr)
# use plot_grid
library(cowplot)
# 处理中文
library(showtext)
showtext_auto()

## DEBUG
# library(httpgd)
# hgd()

args <- commandArgs(TRUE)

work_dir <- args[1]

setwd(work_dir)

# load info --------------------------------------------------------------------
info <- read.table("info.txt", header = TRUE, stringsAsFactors = FALSE)

# ------------------------------------------------------------------------------
# 长度分布
# ------------------------------------------------------------------------------
## *.SeqResult.txt -> b[name,length] -------------------------------------------
data_frames_list <- list()
for (path in dir(pattern = "*.histogram.txt")) {
    message("load ", path)
    df <- data.frame(name = strsplit(path, "[.]")[[1]][1])
    if (file.info(path)$size > 0) {
        data_frames_list[[path]] <-
            cbind(df, read.table(path, header = TRUE, stringsAsFactors = FALSE))
    } else {
        message("skip ", path, " for empty!")
    }
}
b <- do.call(rbind, data_frames_list)

## histogram.pdf ---------------------------------------------------------------

pdf("histogram.pdf", width = 16, height = 9)

p <- ggplot(b, aes(x = length, group = name, weight = weight)) +
    geom_histogram(binwidth = 1) +
    facet_wrap(~name, scales = "free")
print(p)

p <- ggplot(b, aes(x = length, group = name, weight = weight)) +
    geom_histogram(binwidth = 1) +
    scale_y_log10() +
    facet_wrap(~name, scales = "free")
print(p)

for (name in unique(b$name)) {
    print(name)
    c <- b[b$name == name, ]

    p1 <- ggplot(c, aes(x = length, group = name, weight = weight)) +
        geom_histogram(binwidth = 1) +
        theme(text = element_text(size = 20)) +
        facet_wrap(~name, scales = "free")

    p2 <- ggplot(c, aes(x = length, group = name, weight = weight)) +
        geom_histogram(binwidth = 1) +
        scale_y_log10() +
        theme(text = element_text(size = 20)) +
        facet_wrap(~name, scales = "free")

    p <- plot_grid(p1, p2, nrow = 2)
    print(p)
}

dev.off()
## END -------------------------------------------------------------------------

# END --------------------------------------------------------------------------
