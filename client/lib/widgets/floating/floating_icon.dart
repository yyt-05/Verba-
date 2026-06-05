import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';

class FloatingIcon extends StatelessWidget {
  final String name;
  final double size;

  const FloatingIcon({super.key, required this.name, this.size = 24});

  @override
  Widget build(BuildContext context) {
    return SvgPicture.asset(
      'assets/icons/$name.svg',
      width: size,
      height: size,
    );
  }
}
