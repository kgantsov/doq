import { Flex, Box, Heading } from "@chakra-ui/react";
import { Link as RouterLink } from "react-router-dom";
import { ColorModeButton } from "./ui/color-mode";

const Navbar = () => {
  return (
    <header>
      <Flex padding="10px" alignItems="center">
        <Box>
          <Heading size="lg">
            <RouterLink to="/">DOQ</RouterLink>
          </Heading>
        </Box>
        <Flex marginLeft="auto">
          <ColorModeButton />
        </Flex>
      </Flex>
    </header>
  );
};

export default Navbar;
