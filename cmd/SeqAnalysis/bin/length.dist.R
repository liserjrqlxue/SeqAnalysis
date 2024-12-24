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
data_frames_list2 <- list()
for (path in dir(pattern = "*.histogram.txt")) {
    name <- strsplit(path, "[.]")[[1]][1]
    message("load ", name, ":", path)
    seq_length <- str_length((info[info$id == name, ]$seq))
    df <- data.frame(name = name, seq_length = seq_length)
    h <- read.table(path, header = TRUE, stringsAsFactors = FALSE)
    message("seq length: ", seq_length)
    h2 <- h[h$length < seq_length - 5 | h$length > seq_length + 5, ]
    if (file.info(path)$size > 0) {
        data_frames_list[[path]] <- cbind(df, h)
        data_frames_list2[[path]] <- cbind(df, h2)
    } else {
        message("skip ", path, " for empty!")
    }
}
b <- do.call(rbind, data_frames_list)
b2 <- do.call(rbind, data_frames_list2)

b$name <- factor(b$name, levels = info$id)
b2$name <- factor(b2$name, levels = info$id)

## histogram.pdf ---------------------------------------------------------------

pdf("histogram.pdf", width = 16, height = 9)

p1 <- ggplot(b, aes(x = length, group = name, weight = weight)) +
    geom_histogram(binwidth = 1) +
    geom_vline(xintercept = 5:9 * 10, col = "red") +
    scale_x_continuous(breaks = seq(0, 100, by = 5))

p2 <- ggplot(b2, aes(x = length, group = name, weight = weight)) +
    geom_histogram(binwidth = 1) +
    facet_grid(vars(seq_length)) +
    geom_vline(xintercept = 5:9 * 10, col = "red") +
    scale_x_continuous(breaks = seq(0, 100, by = 5))

p1_grid <- p1 + facet_grid(vars(seq_length))
p1_grid_ylog10 <- p1_grid + scale_y_log10()
p1_wrap <- p1 + facet_wrap(vars(name), ncol = 3)
p1_wrap_ylog10 <- p1_wrap + scale_y_log10()


p2_grid <- p2 + facet_grid(vars(seq_length))
p2_grid_ylog10 <- p2_grid + scale_y_log10()
p2_wrap <- p2 + facet_wrap(vars(name), ncol = 3)
p2_wrap_ylog10 <- p2_wrap + scale_y_log10()

print(p1_grid)
print(p1_grid_ylog10)
print(p2_grid)
print(p2_grid_ylog10)

p <- plot_grid(p1_grid, p1_grid_ylog10, ncol = 2)
print(p)
p <- plot_grid(p2_grid, p2_grid_ylog10, ncol = 2)
print(p)



p <- plot_grid(
    p1_grid, p1_grid_ylog10,
    p2_grid, p2_grid_ylog10,
    ncol = 2
)
print(p)

print(p1_wrap)
print(p1_wrap_ylog10)
print(p2_wrap)
print(p2_wrap_ylog10)

for (name in info$id) {
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

    c <- b2[b2$name == name, ]
    p3 <- ggplot(c, aes(x = length, group = name, weight = weight)) +
        geom_histogram(binwidth = 1) +
        theme(text = element_text(size = 20)) +
        facet_wrap(~name, scales = "free")

    p4 <- ggplot(c, aes(x = length, group = name, weight = weight)) +
        geom_histogram(binwidth = 1) +
        scale_y_log10() +
        theme(text = element_text(size = 20)) +
        facet_wrap(~name, scales = "free")

    p <- plot_grid(p3, p4, nrow = 2)
    print(p)
    p <- plot_grid(p1, p2, p3, p4, nrow = 2)
    print(p)
}

dev.off()
## END -------------------------------------------------------------------------

# END --------------------------------------------------------------------------
