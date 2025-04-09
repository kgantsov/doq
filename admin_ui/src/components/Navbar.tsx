import { Flex, Box, Heading, Button } from "@chakra-ui/react";
import { Link as RouterLink } from "react-router-dom";

const Navbar = () => {
  return (
    <Flex bg="teal.500" color="white" padding="10px" alignItems="center">
      <Box>
        <Heading size="lg">
          <RouterLink to="/">DOQ</RouterLink>
        </Heading>
      </Box>
      <Flex marginLeft="auto"></Flex>
    </Flex>
  );
};

export default Navbar;
