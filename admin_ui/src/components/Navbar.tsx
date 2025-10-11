import { Flex, Box, Heading, Stack, Spacer } from "@chakra-ui/react";
import { NavLink as RouterLink } from "react-router-dom";
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

        <Stack
          direction={{ base: "column", md: "row" }}
          gap={5}
          marginLeft={30}
          marginRight={30}
        >
          <RouterLink
            to="/"
            className={({ isActive, isPending }) =>
              isPending
                ? "navigation pending"
                : isActive
                  ? "navigation active"
                  : "navigation"
            }
          >
            Queues
          </RouterLink>
          <RouterLink
            to="/servers"
            className={({ isActive, isPending }) =>
              isPending
                ? "navigation pending"
                : isActive
                  ? "navigation active"
                  : "navigation"
            }
          >
            Servers
          </RouterLink>
        </Stack>

        <Spacer />

        <Flex marginLeft="auto">
          <ColorModeButton />
        </Flex>
      </Flex>
    </header>
  );
};

export default Navbar;
