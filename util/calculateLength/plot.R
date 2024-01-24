read.table("20_0.99_0.5_0.9.txt.histogram.txt") -> a
colnames(a) <- c("length", "Probability")
library(ggplot2)
ggplot(a, aes(length, Probability)) +
    geom_point() +
    geom_line()
ggplot(a, aes(length, Probability)) +
    geom_point() +
    geom_line() +
    scale_y_log10()

a$count <- a$Probability * 1e5

library(httpgd)
hgd()
ggplot(a, aes(length, count)) +
    geom_point() +
    geom_line() +
    scale_y_log10()
