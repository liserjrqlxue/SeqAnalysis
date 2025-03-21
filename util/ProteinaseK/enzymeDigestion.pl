#!/usr/bin/perl

use strict;
use warnings;

# // 脂肪氨基酸
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

# // 芳香族胺基酸
my%AromaticAminoAcids=(
    'F'=>'Phe', # 苯丙氨酸
    'W'=>'Trp', # 色氨酸
    'Y'=>'Tyr', # 色氨酸
);
#!/usr/bin/perl

# ANSI 转义码
# my $color_bg ="\e[47m";
my $color_bg ="\e[1m\e[40m";
my $color_red = "$color_bg\e[91m";      # 红色前景色
my $color_green = "$color_bg\e[92m";    # 绿色前景色
my $color_yellow = "$color_bg\e[93m";    # 绿色前景色
my $color_blue = "$color_bg\e[94m";    # 绿色前景色
my $color_white = "$color_bg\e[97m";      # 红色前景色
my $color_reset = "\e[0m";     # 重置颜色


my$tag=0;
my$count=0;
my$break=100;
while(<>){
    if(/>/){
        print "\n$_";
        $tag=0;
        $count=0;
        next;
    }
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
            print "${color_white}$_${color_reset}";
        }
        if ($count%$break==0){
            print "\n";
        }
    }
    # print "\n";
    # $count=0;
}
print "\n";


sub digestion(){
    my$count=0;
    my$seq=shift;

    my@c=split //,$seq;
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
}



