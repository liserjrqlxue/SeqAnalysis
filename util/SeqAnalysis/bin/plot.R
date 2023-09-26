# /usr/bin/Rscript
library(ggplot2)
library(stringr)
# use plot_grid
library(cowplot)
# 处理中文
library(showtext)
showtext_auto()

args <- commandArgs(TRUE)

work_dir <- args[1]

setwd(work_dir)
setwd("result")

# load info --------------------------------------------------------------------
info <- read.table("info.txt", header = TRUE, stringsAsFactors = FALSE)

# ------------------------------------------------------------------------------
# 错误率分布
# ------------------------------------------------------------------------------

## *.one.step.accuracy.rate.txt -> a[id,pos,rate] ------------------------------
data_frames_list <- list()
for (path in dir(pattern = "*.one.step.accuracy.rate.txt")) {
    data <- read.table(path)
    data_frames_list[[path]] <- data
}
a <- do.call(rbind, data_frames_list)
colnames(a) <- c("id", "tag1", "tag2", "pos", "rate")
a$id <- as.factor(a$id)

a$lab <- str_split_i(a$id, "[-]", 1)

## info -> a$seq ---------------------------------------------------------------
a$seq <- ""
for (id in unique(a$id)) {
    a[a$id == id, ]$seq <- info[info$id == id, ]$seq
}

## ErrRate.pdf -----------------------------------------------------------------
pdf("ErrRate.pdf", width = 16, height = 9)

p <- ggplot(a, aes(as.factor(pos), 1 - rate, group = id, col = id)) +
    geom_point() +
    geom_line() +
    geom_text(label = a$tag2, aes(y = -0.01)) +
    theme(text = element_text(size = 20)) +
    xlab("合成") +
    ylab("error rate") +
    facet_wrap(~seq, ncol = 1)
print(p)

p <- ggplot(a, aes(as.factor(pos), 1 - rate, group = id, col = id)) +
    geom_point() +
    geom_line() +
    geom_text(label = a$tag2, aes(y = -0.01)) +
    theme(text = element_text(size = 20)) +
    xlab("合成") +
    ylab("error rate") +
    facet_wrap(~seq, ncol = 1, scales = "free_y")
print(p)

p <- ggplot(a, aes(as.factor(pos), 1 - rate, group = id, col = id)) +
    geom_point() +
    geom_line() +
    geom_text(label = a$tag2, aes(y = -0.01)) +
    theme(text = element_text(size = 20)) +
    xlab("合成") +
    ylab("error rate") +
    facet_wrap(~lab, ncol = 1)
print(p)

p <- ggplot(a, aes(as.factor(pos), 1 - rate, group = id, col = id)) +
    geom_point() +
    geom_line() +
    geom_text(label = a$tag2, aes(y = -0.01)) +
    theme(text = element_text(size = 20)) +
    xlab("合成") +
    ylab("error rate") +
    facet_wrap(~lab, ncol = 1, scales = "free_y")
print(p)

for (lab in unique(a$lab)) {
    t <- a[a$lab == lab, ]
    print(lab)
    p <- ggplot(t, aes(as.factor(pos), 1 - rate, group = id, col = id)) +
        geom_point() +
        geom_line() +
        geom_text(label = t$tag2, aes(y = -0.01)) +
        theme(text = element_text(size = 20)) +
        xlab("合成") +
        ylab("error rate") +
        ggtitle(lab)
    print(p)
}

for (id in unique(a$id)) {
    t <- a[a$id == id, ]
    print(id)
    p <- ggplot(t, aes(as.factor(pos), 1 - rate, group = id, col = id)) +
        geom_point() +
        geom_line() +
        geom_text(label = t$tag2, aes(y = -0.01)) +
        theme(text = element_text(size = 20)) +
        xlab("合成") +
        ylab("error rate") +
        ggtitle(id)
    print(p)
}




dev.off()
## END -------------------------------------------------------------------------

# ------------------------------------------------------------------------------
# 长度分布
# ------------------------------------------------------------------------------
## *.SeqResult.txt -> b[name,length] -------------------------------------------
data_frames_list <- list()
for (path in dir(pattern = "*.histogram.txt")) {
    message("load ", path)
    df <- data.frame(name = strsplit(path, "[.]")[[1]][1])
    if (file.info(path)$size > 0) {
        data_frames_list[[path]] <- cbind(df,read.table(path,header=T,stringsAsFactors = F))
    } else {
        message("skip ", path, " for empty!")
    }
}
b <- do.call(rbind, data_frames_list)

## histogram.pdf ---------------------------------------------------------------

pdf("histogram.pdf", width = 16, height = 9)

p <- ggplot(b, aes(x = length, group = name, weight=weight)) +
    geom_histogram(binwidth = 1) +
    facet_wrap(~name, scales = "free")
print(p)

p <- ggplot(b, aes(x = length, group = name, weight=weight)) +
    geom_histogram(binwidth = 1) +
    scale_y_log10() +
    facet_wrap(~name, scales = "free")
print(p)

for (name in unique(b$name)) {
    print(name)

    p1 <- ggplot(b[b$name == name, ], aes(x = length, group = name, weight=weight)) +
        geom_histogram(binwidth = 1) +
        theme(text = element_text(size = 20)) +
        facet_wrap(~name, scales = "free")

    p2 <- ggplot(b[b$name == name, ], aes(x = length, group = name, weight=weight)) +
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
