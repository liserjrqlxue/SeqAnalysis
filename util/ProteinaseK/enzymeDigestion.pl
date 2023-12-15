#!/usr/bin/perl

use strict;
use warnings;

my%AliphaticAminoAcids=(
    'A'=>'Ala', # 丙氨酸
    'C'=>'Cys', # 半胱氨酸
    'D'=>'Asp', # 天冬氨酸
    'E'=>'Glu', # 谷氨酸
    'G'=>'Gly', # 甘氨酸
    'I'=>'Ile', # 异亮氨酸
    'K'=>'Lys', # 赖氨酸
    'L'=>'Leu', # 亮氨酸
    'M'=>'Met', # 蛋氨酸
    'N'=>'Asn', # 天冬酰胺
    'Q'=>'Gln', # 谷氨酰胺
    'R'=>'Arg', # 精氨酸
    'S'=>'Ser', # 丝氨酸
    'T'=>'Thr', # 苏氨酸
    'U'=>'Sec', # 硒半胱氨酸
    'V'=>'Val', # 缬氨酸
);
    # 'H'=>'His', # 组氨酸
    # 'P'=>'Pro', # 脯氨酸

my%AromaticAminoAcids=(
    'F'=>'Phe', # 苯丙氨酸
    'W'=>'Trp', # 色氨酸
    'Y'=>'Tyr', # 色氨酸
);
#!/usr/bin/perl

# ANSI 转义码
my $color_red = "\e[40m\e[31m";      # 红色前景色
my $color_green = "\e[40m\e[32m";    # 绿色前景色
my $color_yellow = "\e[40m\e[33m";    # 绿色前景色
my $color_blue = "\e[40m\e[34m";    # 绿色前景色
my $color_reset = "\e[0m\e[0m";     # 重置颜色


my$tag=0;
my$break=80;
my$count=0;
while(<>){
    chomp;
    my@c=split //,$_;
    for(@c){
        $count++;
        if(exists $AliphaticAminoAcids{$_}){
            if($tag+1==0){
                print "${color_red}|${color_reset}";
                if ($count%$break==0){
                   print "\n";
                }
                $count++;
            }
            $tag=1;
            print "${color_yellow}$_${color_reset}";
        }elsif(exists $AromaticAminoAcids{$_}){
            if($tag-1==0){
                print "${color_red}|${color_reset}";
                if ($count%$break==0){
                  print "\n";
                }
                $count++;
            }
            $tag=-1;
            print "${color_green}$_${color_reset}";
        }else{
            $tag=0;
            print "\e[40m$_\e[0m";
        }
        if ($count%$break==0){
            print "\n";
        }
    }
    print "\n";
    $count=0;
}



