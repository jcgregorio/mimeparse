package MIMEParse;
use strict;
use warnings;

our $VERSION = '0.1.2';

=head1 NAME

MIMEParse - MIME-Type Parser

=head1 SYNOPSIS

    use MIMEParse qw( best_match );

    print best_match(['application/xbel+xml', 'text/xml'], 'text/*;q=0.5,*/*; q=0.1');
    # text/xml

=head1 DESCRIPTION

This module provides basic functions for handling mime-types. It can
handle matching mime-types against a list of media-ranges. See section
14.1 of the HTTP specification [RFC 2616] for a complete explanation.

L<http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.1>

This is a port of Joe Gregorio's mimeparse.py, which can be found at
L<http://code.google.com/p/mimeparse/>.

=head1 FUNCTIONS

The following functions are exported on request:

=over

=cut

use Exporter qw( import );
use Scalar::Util qw( looks_like_number );

our @EXPORT_OK = qw( parse_mime_type parse_media_range quality
                     fitness_and_quality_parsed quality_parsed best_match );

# utility function
sub _strip {
    my $s = shift;
    $s =~ s/^\s*//;
    $s =~ s/\s*$//;
    return $s;
}

=item B<parse_mime_type($mime_type)>

Carves up a mime-type and returns a list of the ($type, $subtype,
\%params) where %params is a hash of all the parameters for the
media range.

For example, the media range 'application/xhtml;q=0.5' would get
parsed into:

    ('application', 'xhtml', { q => 0.5 })

=cut

sub parse_mime_type {
    my @parts = split /;/, shift;
    my %params = map { _strip($_) } map { split /=/, $_, 2 } @parts[1..$#parts];
    my $full_type = _strip($parts[0]);

    # Java URLConnection class sends an Accept header that includes a single "*"
    # Turn it into a legal wildcard.
    $full_type = '*/*' if $full_type eq '*';
    my ($type, $subtype) = split qr{/}, $full_type;
    return _strip($type), _strip($subtype), \%params;
}

=item B<parse_media_range($range)>

Carves up a media range and returns a list of the ($type, $subtype,
\%params) where %params is a hash of all the parameters for the
media range.

For example, the media range 'application/*;q=0.5' would get
parsed into:

    ('application', '*', { q => 0.5 })

In addition this function also guarantees that there is a value for 'q'
in the %params hash, filling it in with a proper default if necessary.

=cut

sub parse_media_range {
    my ($type, $subtype, $params) = parse_mime_type(shift);
    if ( not exists $params->{q} or not $params->{q}
            or not looks_like_number($params->{q})
            or $params->{q} > 1 or $params->{q} < 0 ) {
        $params->{q} = 1;
    }
    return $type, $subtype, $params;
}

=item B<fitness_and_quality_parsed($mime_type, @parsed_ranges)>

Find the best match for a given mime-type against a list of media_ranges
that have already been parsed by parse_media_range(). Returns a list of
the fitness value and the value of the 'q' quality parameter of the best
match, or (-1, 0) if no match was found. Just as for quality_parsed(),
@parsed_ranges must be a list of parsed media ranges.

=cut

sub fitness_and_quality_parsed {
    my ($mime_type, @parsed_ranges) = @_;
    my $best_fitness = -1;
    my $best_fit_q = 0;
    my ($target_type, $target_subtype, $target_params)
        = parse_media_range($mime_type);
    while ( my ($type, $subtype, $params) = @{ shift @parsed_ranges || [] } ) {
        if (($type eq $target_type or $type eq '*' or $target_type eq '*')
             and ($subtype eq $target_subtype or $subtype eq '*' or $target_subtype eq '*') ) {
            my $param_matches
                = scalar grep { $_ ne 'q' and exists $params->{$_}
                                and $target_params->{$_} eq $params->{$_} }
                              keys %{$target_params};
            my $fitness = $type eq $target_type ? 100 : 0;
            $fitness += $subtype eq $target_subtype ? 10 : 0;
            $fitness += $param_matches;
            if ($fitness > $best_fitness) {
                $best_fitness = $fitness;
                $best_fit_q = $params->{q};
            }
        }
    }
    return $best_fitness, $best_fit_q;
}

=item B<quality_parsed($mime_type, @parsed_ranges)>

Find the best match for a given mime-type against a list of media_ranges
that have already been parsed by parse_media_range(). Returns the 'q'
quality parameter of the best match, 0 if no match was found. This
function behaves the same as quality() except that @parsed_ranges must
be a list of parsed media ranges.

=cut

sub quality_parsed {
    return ( fitness_and_quality_parsed(@_) )[1];
}

=item B<quality($mime_type, $ranges)>

Returns the quality 'q' of a mime-type when compared against the
media-ranges in $ranges. For example:

    print quality('text/html', 'text/*;q=0.3, text/html;q=0.7, text/html;level=1, text/html;level=2;q=0.4, */*;q=0.5');
    # 0.7

=cut

sub quality {
    my ($mime_type, $ranges) = @_;
    my @parsed_ranges = map { [parse_media_range($_)] } split /,/, $ranges;
    return quality_parsed($mime_type, @parsed_ranges);
}

=item B<best_match(\@supported, $header);>

Takes an arrayref of supported mime-types and finds the best match for
all the media-ranges listed in $header. The value of $header must be a
string that conforms to the format of the HTTP Accept: header. The value
of @supported is a list of mime-types.

    print best_match(['application/xbel+xml', 'text/xml'], 'text/*;q=0.5,*/*; q=0.1');
    # text/xml

=cut

sub best_match {
    my ($supported, $header) = @_;
    my @parsed_header = map { [parse_media_range($_)] } split /,/, $header;
    my @weighted_matches
        = sort { $a->[0][0] <=> $b->[0][0] || $a->[0][1] <=> $b->[0][1] }
               map { [ [fitness_and_quality_parsed($_, @parsed_header)], $_ ] }
                   @{$supported};
    return $weighted_matches[-1][0][1] ? $weighted_matches[-1][1] : '';
}

=back

=head1 AUTHORS

    Joe Gregorio <joe@bitworking.org>
    Stanis Trendelenburg <stanis.trendelenburg@gmail.com> (Perl port)

=cut

return 1 if caller; # magic return value when used as a module

require Test::More;

Test::More->import(tests => 24);

is_deeply( [parse_media_range('application/xml;q=1')],
    ['application', 'xml', { q => 1 }] );
is_deeply( [parse_media_range('application/xml')],
    ['application', 'xml', { q => 1 }] );
is_deeply( [parse_media_range('application/xml;q=')],
    ['application', 'xml', { q => 1 }] );
is_deeply( [parse_media_range('application/xml ; q=')],
    ['application', 'xml', { q => 1 }] );
is_deeply( [parse_media_range('application/xml ; q=1;b=other')],
    ['application', 'xml', { q => 1, b => 'other' }] );
is_deeply( [parse_media_range('application/xml ; q=2;b=other')],
    ['application', 'xml', { q => 1, b => 'other' }] );

# Java URLConnection class sends an Accept header that includes a single *
is_deeply( [parse_media_range(" *; q=.2")], ['*', '*', { q => '.2' }] );

# example from rfc 2616
my $accept = "text/*;q=0.3, text/html;q=0.7, text/html;level=1, text/html;level=2;q=0.4, */*;q=0.5";
is( quality("text/html;level=1", $accept), 1   );
is( quality("text/html",         $accept), 0.7 );
is( quality("text/plain",        $accept), 0.3 );
is( quality("image/jpeg",        $accept), 0.5 );
is( quality("text/html;level=2", $accept), 0.4 );
is( quality("text/html;level=3", $accept), 0.7 );

my $mime_types_supported = ['application/xbel+xml', 'application/xml'];
is( best_match($mime_types_supported, 'application/xbel+xml'),
    'application/xbel+xml', "direct match" );
is( best_match($mime_types_supported, 'application/xbel+xml; q=1'),
    'application/xbel+xml', "direct match with a q parameter" );
is( best_match($mime_types_supported, 'application/xml; q=1'),
    'application/xml', "direct match of our second choice with a q parameter" );
is( best_match($mime_types_supported, 'application/*; q=1'),
    'application/xml', "match using a subtype wildcard" );
is( best_match($mime_types_supported, '*/*'),
    'application/xml', "match using a type wildcard" );

$mime_types_supported = ['application/xbel+xml', 'text/xml'];
is( best_match($mime_types_supported, 'text/*;q=0.5,*/*; q=0.1'),
    'text/xml', "match using a type versus a lower weighted subtype" );
is( best_match($mime_types_supported, 'text/html,application/atom+xml; q=0.9'),
    '', "fail to match anything" );

$mime_types_supported = ['application/json', 'text/html'];
is( best_match($mime_types_supported, 'application/json, text/javascript, */*'),
    'application/json', "common AJAX scenario" );
is( best_match($mime_types_supported, 'application/json, text/html;q=0.9'),
    'application/json', "verify fitness ordering" );

$mime_types_supported = ['image/*', 'application/xml'];
is( best_match($mime_types_supported, 'image/png'),
    'image/*', "match using a type wildcard" );
is( best_match($mime_types_supported, 'image/*'),
    'image/*', "match using a wildcard for both requested and supported " );

